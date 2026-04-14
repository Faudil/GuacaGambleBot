import discord
from discord.ext import commands
from discord.ui import View, Button, Select
import random
import datetime

from src.database.item import add_item_to_inventory, has_item, remove_item_from_inventory, get_all_user_inventory
from src.database.job import add_job_xp, get_job_data
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.pets import get_active_pet
from src.database.farming import get_user_plots, plant_seed, harvest_plot
from src.items.FarmItem import Seed
from src.items.LandDeed import VegetablePatchDeed, GreenhouseDeed, OrchardDeed
from src.models.Pet import PetBonus
from src.database.settings import get_language
from src.utils.i18n import t, get_item_name
from src.globals import ITEMS_REGISTRY


class SeedSelect(Select):
    def __init__(self, ctx, zone_key, plot_index, seeds, lang):
        self.ctx = ctx
        self.zone_key = zone_key
        self.plot_index = plot_index
        self.lang = lang
        options = [
            discord.SelectOption(
                label=f"{s.display_name(lang)}",
                value=s.name,
                description=s.display_description(lang)
            ) for s in seeds
        ]
        super().__init__(placeholder=t("farm.select_seed_placeholder", lang), options=options)

    async def callback(self, interaction: discord.Interaction):
        if interaction.user != self.ctx.author: return
        seed_name = self.values[0]
        seed = ITEMS_REGISTRY.get(seed_name)
        if not seed or not isinstance(seed, Seed): return

        level, _ = get_job_data(interaction.user.id, "farmer")
        reduction = min(0.5, level * 0.01)
        final_grow_time = int(seed.grow_time * (1 - reduction))

        plant_seed(interaction.user.id, self.zone_key, self.plot_index, seed.name, final_grow_time)
        remove_item_from_inventory(interaction.user.id, seed.name)

        embed = discord.Embed(
            title=t("farm.planted_title", self.lang),
            description=t("farm.planted_desc", self.lang, item=seed.display_name(self.lang), time=final_grow_time // 60),
            color=discord.Color.green()
        )
        await interaction.response.edit_message(embed=embed, view=None)


class FarmPlotView(View):
    def __init__(self, ctx, zone_key, zone_name, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.zone_key = zone_key
        self.zone_name = zone_name
        self.lang = lang
        self.refresh_plots()

    def refresh_plots(self):
        self.clear_items()
        plots = get_user_plots(self.ctx.author.id, self.zone_key)
        plot_data = {p['plot_index']: p for p in plots}

        for i in range(3):
            data = plot_data.get(i)
            if not data:
                btn = Button(label=t("farm.plot_empty", self.lang, idx=i+1), style=discord.ButtonStyle.secondary, custom_id=f"plot_{i}")
                btn.callback = self.make_plant_callback(i)
            else:
                plant_time = datetime.datetime.fromisoformat(data['plant_time']) if isinstance(data['plant_time'], str) else data['plant_time']
                elapsed = (datetime.datetime.now() - plant_time).total_seconds()
                grow_time = data['grow_time']
                
                if elapsed >= grow_time:
                    btn = Button(label=t("farm.plot_ready", self.lang, item=get_item_name(data['item_name'], self.lang)), 
                                 style=discord.ButtonStyle.success, emoji="✨", custom_id=f"plot_{i}")
                    btn.callback = self.make_harvest_callback(i, data)
                else:
                    percent = min(100, int((elapsed / grow_time) * 100))
                    btn = Button(label=t("farm.plot_growing", self.lang, item=get_item_name(data['item_name'], self.lang), pc=percent), 
                                 style=discord.ButtonStyle.primary, disabled=True, custom_id=f"plot_{i}")
            self.add_item(btn)

    def make_plant_callback(self, idx):
        async def callback(interaction: discord.Interaction):
            if interaction.user != self.ctx.author: return
            inv = get_all_user_inventory(interaction.user.id)
            seeds = []
            for item in inv:
                obj = ITEMS_REGISTRY.get(item['name'].lower())
                if isinstance(obj, Seed):
                    seeds.append(obj)
            
            if not seeds:
                return await interaction.response.send_message(t("farm.no_seeds", self.lang), ephemeral=True)
            view = View()
            view.add_item(SeedSelect(self.ctx, self.zone_key, idx, seeds, self.lang))
            await interaction.response.edit_message(content=t("farm.choose_seed", self.lang), view=view)
        return callback

    def make_harvest_callback(self, idx, data):
        async def callback(interaction: discord.Interaction):
            if interaction.user != self.ctx.author: return
            
            seed = ITEMS_REGISTRY.get(data['item_name'])
            crop_class = seed.crop_class if seed and hasattr(seed, 'crop_class') else None
            if not crop_class: return
            
            crop = crop_class()
            level, _ = get_job_data(interaction.user.id, "farmer")
            pet = get_active_pet(interaction.user.id)
            
            pet_bonus = pet.level if (pet and pet.bonus == PetBonus.FARM) else 0
            # Yield: 1 base + bonus from pet
            quantity = 1 + (pet_bonus // 10)
            # Small extra chance based on level
            if random.random() < (level * 0.02):
                quantity += 1

            xp_gain = (int(crop.price * 0.8) + 10) * quantity
            add_item_to_inventory(interaction.user.id, crop.name, quantity)
            increment_stat(interaction.user.id, "items_farmed", quantity)
            add_job_xp(interaction.user.id, "farmer", xp_gain)
            harvest_plot(interaction.user.id, self.zone_key, idx)

            embed = discord.Embed(title=f"🏡 {self.zone_name}", color=discord.Color.green())
            msg_loot = t("farm.harvest_msg", self.lang, qty=quantity, item=crop.display_name(self.lang))
            embed.description = t("farm.success_desc", self.lang, loot=msg_loot, value=crop.price * quantity, xp=xp_gain)
            
            await interaction.response.edit_message(content=None, embed=embed, view=None)
            
            unlocks = check_and_unlock_achievements(interaction.user.id)
            if unlocks:
                await interaction.channel.send(content=interaction.user.mention, embed=format_achievements_unlocks(unlocks, self.lang))
        return callback


class FarmDashboardView(View):
    def __init__(self, ctx, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.lang = lang

    async def check_land_and_go(self, interaction, deed, zone_key):
        if interaction.user != self.ctx.author: return

        if deed and not has_item(self.ctx.author.id, deed.name):
            embed = discord.Embed(
                title=t("farm.no_land_title", self.lang),
                description=t("farm.no_land_desc", self.lang, deed=deed.display_name(self.lang)),
                color=discord.Color.red()
            )
            return await interaction.response.send_message(embed=embed, ephemeral=True)

        zone_name = t(f"farm.{zone_key}_name", self.lang)
        embed = discord.Embed(
            title=f"🏡 {zone_name}",
            description=t("farm.zone_welcome", self.lang),
            color=discord.Color.gold()
        )
        view = FarmPlotView(self.ctx, zone_key, zone_name, self.lang)
        return await interaction.response.edit_message(embed=embed, view=view)

    @discord.ui.button(label="public_placeholder", style=discord.ButtonStyle.secondary, emoji="🌾", row=0)
    async def public_zone(self, interaction: discord.Interaction, button: Button):
        await self.check_land_and_go(interaction, None, "public")

    @discord.ui.button(label="veggie_placeholder", style=discord.ButtonStyle.primary, emoji="🥕", row=0)
    async def veggie_zone(self, interaction: discord.Interaction, button: Button):
        await self.check_land_and_go(interaction, VegetablePatchDeed(), "veggie")

    @discord.ui.button(label="greenhouse_placeholder", style=discord.ButtonStyle.success, emoji="🌡️", row=1)
    async def greenhouse_zone(self, interaction: discord.Interaction, button: Button):
        await self.check_land_and_go(interaction, GreenhouseDeed(), "greenhouse")

    @discord.ui.button(label="orchard_placeholder", style=discord.ButtonStyle.danger, emoji="✨", row=1)
    async def orchard_zone(self, interaction: discord.Interaction, button: Button):
        await self.check_land_and_go(interaction, OrchardDeed(), "orchard")


class Farm(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='farm')
    async def farm(self, ctx):
        """Gestion agricole. Plante tes graines et récolte tes cultures."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        embed = discord.Embed(
            title=t("farm.map_title", lang),
            description=t("farm.map_desc", lang),
            color=discord.Color.dark_green()
        )
        view = FarmDashboardView(ctx, lang)
        for item in view.children:
            if isinstance(item, Button):
                if item.label == "public_placeholder": item.label = t("farm.public_label", lang)
                if item.label == "veggie_placeholder": item.label = t("farm.veggie_label", lang)
                if item.label == "greenhouse_placeholder": item.label = t("farm.greenhouse_label", lang)
                if item.label == "orchard_placeholder": item.label = t("farm.orchard_label", lang)

        await ctx.send(embed=embed, view=view)


async def setup(bot):
    await bot.add_cog(Farm(bot))