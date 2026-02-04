import discord
from discord.ext import commands
from discord.ui import View, Button
import random

from src.command_decorators import daily_limit
from src.database.item import add_item_to_inventory, has_item
from src.database.job import add_job_xp, get_job_data
from src.items.FarmItem import (
    Wheat, Oat, Corn, Potato, Tomato, Pumpkin,
    CoffeeBean, CocoaBean, Strawberry, GoldenApple, StarFruit,
)


class FarmActionView(View):
    def __init__(self, ctx, zone_name, potential_loots):
        super().__init__(timeout=15)
        self.ctx = ctx
        self.zone_name = zone_name
        self.loots = potential_loots

    @discord.ui.button(label="🚜 Exploiter ce terrain", style=discord.ButtonStyle.success)
    async def work(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        level, _ = get_job_data(self.ctx.author.id, "farmer")
        weight_common = max(10, 50 - level)
        weight_uncommon = 30
        weight_rare = min(60, 20 + level)
        weights = [weight_common, weight_uncommon, weight_rare][:len(self.loots)]
        loot = random.choices(self.loots, weights=weights, k=1)[0]
        double_drop_chance = min(50, level * 2)
        quantity = 1
        is_double = False
        if random.randint(1, 100) <= double_drop_chance:
            quantity = 2
            is_double = True
        xp_gain = (int(loot.price * 0.6) + 5) * quantity
        add_item_to_inventory(self.ctx.author.id, loot.name,
                              quantity)
        add_job_xp(self.ctx.author.id, "farmer", xp_gain)
        embed = discord.Embed(title=f"🏡 {self.zone_name}", color=discord.Color.green())
        msg_loot = f"📦 Récolte : **{quantity}x {loot.name}**"
        if is_double:
            msg_loot += f" 🔥 **DOUBLE RÉCOLTE !** (Bonus Niv.{level})"
        embed.description = (
            f"{msg_loot}\n"
            f"💰 Valeur estimée : **${loot.price * quantity}**\n"
            f"📈 XP Fermier : +{xp_gain}"
        )
        await interaction.response.edit_message(content=None, embed=embed, view=None)
        self.stop()


class FarmDashboardView(View):
    def __init__(self, ctx):
        super().__init__(timeout=60)
        self.ctx = ctx

    async def check_land_and_go(self, interaction, deed_name, zone_name, loots):
        if interaction.user != self.ctx.author:
            return None

        if deed_name and not has_item(self.ctx.author.id, deed_name):
            embed = discord.Embed(
                title="⛔ Terrain non acquis",
                description=f"Tu n'es pas propriétaire de ce terrain !\nAchète le **{deed_name}** au `!shop`.",
                color=discord.Color.red()
            )
            return await interaction.response.send_message(embed=embed, ephemeral=True)

        embed = discord.Embed(
            title=f"🏡 {zone_name}",
            description="Bienvenue chez toi. Tes cultures sont prêtes.",
            color=discord.Color.gold()
        )
        view = FarmActionView(self.ctx, zone_name, loots)
        return await interaction.response.edit_message(embed=embed, view=view)

    @discord.ui.button(label="Champs Publics", style=discord.ButtonStyle.secondary, emoji="🌾", row=0)
    async def public_zone(self, interaction: discord.Interaction, button: Button):
        loots = [Wheat(), Oat(), Corn()]
        await self.check_land_and_go(interaction, None, "Champs Communaux", loots)

    @discord.ui.button(label="Mon Potager", style=discord.ButtonStyle.primary, emoji="🥕", row=0)
    async def veggie_zone(self, interaction: discord.Interaction, button: Button):
        loots = [Potato(), Tomato(), Pumpkin()]
        await self.check_land_and_go(interaction, "Terrain : Potager", "Potager Privé", loots)

    @discord.ui.button(label="Ma Serre", style=discord.ButtonStyle.success, emoji="🌡️", row=1)
    async def greenhouse_zone(self, interaction: discord.Interaction, button: Button):
        loots = [CoffeeBean(), CocoaBean(), Strawberry()]
        await self.check_land_and_go(interaction, "Terrain : Serre Tropicale", "Serre Tropicale", loots)

    @discord.ui.button(label="Mon Verger", style=discord.ButtonStyle.danger, emoji="✨", row=1)
    async def orchard_zone(self, interaction: discord.Interaction, button: Button):
        loots = [Strawberry(), GoldenApple(), StarFruit()]
        await self.check_land_and_go(interaction, "Terrain : Verger Enchanté", "Verger Céleste", loots)


class Farm(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='farm')
    @daily_limit("farm", 5)
    async def farm(self, ctx):
        embed = discord.Embed(
            title="🚜 Carte de tes Propriétés",
            description="Sélectionne un terrain pour y travailler.\nTu peux acheter de nouvelles parcelles au `!shop`.",
            color=discord.Color.dark_green()
        )
        view = FarmDashboardView(ctx)
        await ctx.send(embed=embed, view=view)


async def setup(bot):
    await bot.add_cog(Farm(bot))