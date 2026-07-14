import discord
from discord.ext import commands
from discord.ui import View, Button
import random
import asyncio
import time

from src.command_decorators import daily_limit, ActivityType, opening_hours
from src.database.item import add_item_to_inventory
from src.database.job import add_job_xp, get_job_data
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.pets import get_active_pet
from src.items.FishingLoot import KrakenTentacle, Swordfish, Pufferfish, Trout, Sardine, OldBoot, Salmon, Carp, Shark, \
    Whale
from src.models.Pet import PetBonus
from src.database.settings import get_language
from src.utils.i18n import t


class FishingGameView(View):
    def __init__(self, ctx, biome_name, time_limit, loot_pool, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.biome_name = biome_name
        self.lang = lang

        lvl, _ = get_job_data(self.ctx.author.id, "fisher")
        pet = get_active_pet(self.ctx.author.id)
        if pet:
            pet_bonus = pet.level // 4 if pet.bonus == PetBonus.FISH else 0
        else:
            pet_bonus = 0
            
        from src.utils.NPCManager import NPCManager
        npc_bonuses = NPCManager.get_user_bonuses(self.ctx.author.id)
        fishing_bonus = npc_bonuses.get("fishing_time_bonus", 0.0)
        
        self.time_limit = time_limit + (lvl + pet_bonus) * 0.1 + fishing_bonus

        self.loot_pool = loot_pool
        self.bite_active = False
        self.start_time = 0.0
        self.message = None

    async def start_game(self, message):
        self.message = message

        wait = random.uniform(2, 5)
        await asyncio.sleep(wait)

        self.bite_active = True
        self.start_time = time.time()

        button = self.children[0]
        button.label = t("fishing.bite_label", self.lang)
        button.style = discord.ButtonStyle.success
        button.emoji = "🎣"

        embed = message.embeds[0]
        embed.color = discord.Color.green()
        embed.description = t("fishing.bite_desc", self.lang, biome=self.biome_name.upper())

        await message.edit(embed=embed, view=self)

        await asyncio.sleep(self.time_limit + 1)

        if self.bite_active:
            self.bite_active = False
            self.stop()

            button.label = t("fishing.too_slow_label", self.lang)
            button.style = discord.ButtonStyle.danger
            button.disabled = True

            embed.description = t("fishing.escaped_desc", self.lang, time=self.time_limit)
            embed.color = discord.Color.red()
            await message.edit(embed=embed, view=self)

    @discord.ui.button(label="wait_placeholder", style=discord.ButtonStyle.secondary, emoji="🌊")
    async def catch_button(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return

        if not self.bite_active:
            self.stop()
            embed = interaction.message.embeds[0]
            embed.description = t("fishing.too_early_desc", self.lang)
            embed.color = discord.Color.orange()
            button.label = t("fishing.too_early_label", self.lang)
            button.disabled = True
            await interaction.response.edit_message(embed=embed, view=self)
            return

        reaction = time.time() - self.start_time
        self.bite_active = False
        self.stop()

        if reaction > self.time_limit:
            embed = interaction.message.embeds[0]
            embed.description = t("fishing.too_slow_reflex", self.lang, reaction=round(reaction, 2), time=self.time_limit)
            embed.color = discord.Color.red()
            button.label = t("fishing.escaped_label", self.lang)
            button.style = discord.ButtonStyle.danger
            button.disabled = True
            await interaction.response.edit_message(embed=embed, view=self)
            return

        loot_item = self.get_random_loot(reaction)
        xp_gain = int(10 + (1.5 / self.time_limit) * 10)

        add_item_to_inventory(self.ctx.author.id, loot_item.name)
        increment_stat(self.ctx.author.id, "items_fished")
        add_job_xp(self.ctx.author.id, "fisher", xp_gain)

        embed = interaction.message.embeds[0]
        embed.title = t("fishing.success_title", self.lang)
        embed.color = discord.Color.gold()
        embed.description = t("fishing.success_desc", self.lang, reaction=f"{reaction:.3f}", item=loot_item.display_name(self.lang), xp=xp_gain)

        button.label = t("fishing.caught_label", self.lang, item=loot_item.display_name(self.lang))
        button.disabled = True

        await interaction.response.edit_message(embed=embed, view=self)
        
        unlocks = check_and_unlock_achievements(self.ctx.author.id)
        if unlocks:
            await interaction.channel.send(content=interaction.user.mention, embed=format_achievements_unlocks(unlocks, self.lang))

    def get_random_loot(self, reaction):
        roll = random.random()
        is_perfect = reaction < (self.time_limit * 0.7)
        if is_perfect and roll < 0.20:
            return self.loot_pool[-1]
        elif roll < 0.60:
            index = random.randint(1, len(self.loot_pool) - 1)
            return self.loot_pool[index]
        else:
            return self.loot_pool[0]


class FishBiomeView(View):
    def __init__(self, ctx, lang):
        super().__init__(timeout=30)
        self.ctx = ctx
        self.lang = lang

    async def launch_biome(self, interaction, biome_key, limit, items):
        if interaction.user != self.ctx.author:
            return

        biome_name = t(f"fishing.{biome_key}_name", self.lang)
        embed = discord.Embed(
            title=t("fishing.direction_title", self.lang, biome=biome_name),
            description=t("fishing.cast_desc", self.lang),
            color=discord.Color.blue()
        )
        game_view = FishingGameView(self.ctx, biome_name, limit, items, self.lang)
        # Update button label
        for item in game_view.children:
            if isinstance(item, Button) and item.label == "wait_placeholder":
                item.label = t("fishing.wait_label", self.lang)

        await interaction.response.edit_message(embed=embed, view=game_view)
        asyncio.create_task(game_view.start_game(interaction.message))

    @discord.ui.button(label="pond_placeholder", style=discord.ButtonStyle.success)
    async def pond(self, interaction: discord.Interaction, button: Button):
        items = [OldBoot(), Trout(), Salmon()]
        await self.launch_biome(interaction, "pond", 2.0, items)

    @discord.ui.button(label="river_placeholder", style=discord.ButtonStyle.primary)
    async def river(self, interaction: discord.Interaction, button: Button):
        items = [Salmon(), Sardine(), Carp(), Pufferfish()]
        await self.launch_biome(interaction, "river", 1.2, items)

    @discord.ui.button(label="ocean_placeholder", style=discord.ButtonStyle.danger)
    async def ocean(self, interaction: discord.Interaction, button: Button):
        items = [Pufferfish(), Swordfish(), Shark(), Whale(), KrakenTentacle()]
        await self.launch_biome(interaction, "ocean", 0.7, items)


class Fishing(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='fish')
    @daily_limit("fish", 10)
    async def fish(self, ctx):
        """Aller à la pêche."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        
        from src.database.housing import is_inventory_full
        full, current, limit = is_inventory_full(ctx.author.id)
        if full:
            return await ctx.send(t("housing.inv_full_warning", lang, current=current, limit=limit))
            
        embed = discord.Embed(
            title=t("fishing.session_title", lang),
            description=t("fishing.session_desc", lang),
            color=discord.Color.teal()
        )
        embed.add_field(name=t("fishing.pond_field_name", lang), value=t("fishing.pond_field_value", lang), inline=True)
        embed.add_field(name=t("fishing.river_field_name", lang), value=t("fishing.river_field_value", lang), inline=True)
        embed.add_field(name=t("fishing.ocean_field_name", lang), value=t("fishing.ocean_field_value", lang), inline=True)
        lvl, _ = get_job_data(ctx.author.id, "fisher")
        embed.set_footer(text=t("fishing.footer", lang, lvl=lvl))
        
        view = FishBiomeView(ctx, lang)
        # Update button labels
        for item in view.children:
            if isinstance(item, Button):
                if item.label == "pond_placeholder": item.label = t("fishing.pond_label", lang)
                if item.label == "river_placeholder": item.label = t("fishing.river_label", lang)
                if item.label == "ocean_placeholder": item.label = t("fishing.ocean_label", lang)

        await ctx.send(embed=embed, view=view)


async def setup(bot):
    await bot.add_cog(Fishing(bot))