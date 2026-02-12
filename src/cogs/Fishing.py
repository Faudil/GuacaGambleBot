import discord
from discord.ext import commands
from discord.ui import View, Button
import random
import asyncio
import time

from src.command_decorators import daily_limit, ActivityType, opening_hours
from src.database.item import add_item_to_inventory
from src.database.job import add_job_xp, get_job_data
from src.items.FishingLoot import KrakenTentacle, Swordfish, Pufferfish, Trout, Sardine, OldBoot, Salmon, Carp, Shark, \
    Whale


class FishingGameView(View):
    def __init__(self, ctx, biome_name, time_limit, loot_pool):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.biome_name = biome_name

        lvl, _ = get_job_data(self.ctx.author.id, "fisher")
        self.time_limit = time_limit + (lvl * 0.1)

        self.loot_pool = loot_pool
        self.bite_active = False
        self.start_time = 0
        self.message = None

    async def start_game(self, message):
        self.message = message

        wait = random.uniform(2, 5)
        await asyncio.sleep(wait)

        self.bite_active = True
        self.start_time = time.time()

        button = self.children[0]
        button.label = "🦈 ÇA MORD ! CLIQUE !"
        button.style = discord.ButtonStyle.success
        button.emoji = "🎣"

        embed = message.embeds[0]
        embed.color = discord.Color.green()
        embed.description = f"## 🌊 {self.biome_name.upper()} : ÇA TIRE !\n**CLIQUE VITE !**"

        await message.edit(embed=embed, view=self)

        await asyncio.sleep(self.time_limit + 1)

        if self.bite_active:
            self.bite_active = False
            self.stop()

            button.label = "Trop lent..."
            button.style = discord.ButtonStyle.danger
            button.disabled = True

            embed.description = f"❌ **Le poisson s'est échappé !**\nIl fallait réagir en moins de {self.time_limit}s."
            embed.color = discord.Color.red()
            await message.edit(embed=embed, view=self)

    @discord.ui.button(label="... Attendre ...", style=discord.ButtonStyle.secondary, emoji="🌊")
    async def catch_button(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return

        if not self.bite_active:
            self.stop()
            embed = interaction.message.embeds[0]
            embed.description = "⚠️ **Tu as tiré trop tôt !** Le poisson a pris peur."
            embed.color = discord.Color.orange()
            button.label = "Raté (Trop tôt)"
            button.disabled = True
            await interaction.response.edit_message(embed=embed, view=self)
            return

        reaction = time.time() - self.start_time
        self.bite_active = False
        self.stop()



        if reaction > self.time_limit:
            embed = interaction.message.embeds[0]
            embed.description = f"🐌 **Trop lent !** ({reaction:.2f}s)\nCe biome demande des réflexes de {self.time_limit}s."
            embed.color = discord.Color.red()
            button.label = "Échappé"
            button.style = discord.ButtonStyle.danger
            button.disabled = True
            await interaction.response.edit_message(embed=embed, view=self)
            return

        loot_item = self.get_random_loot(reaction)
        xp_gain = int(10 + (1.5 / self.time_limit) * 10)

        add_item_to_inventory(self.ctx.author.id, loot_item.name)
        add_job_xp(self.ctx.author.id, "fisher", xp_gain)

        embed = interaction.message.embeds[0]
        embed.title = "🎣 PÊCHE RÉUSSIE !"
        embed.color = discord.Color.gold()
        embed.description = (
            f"⏱️ Réflexe : **{reaction:.3f}s**\n"
            f"🐟 Tu as attrapé : **{loot_item.name}**\n"
            f"✨ XP Pêcheur : +{xp_gain}"
        )

        button.label = f"Attrapé : {loot_item.name}"
        button.disabled = True

        await interaction.response.edit_message(embed=embed, view=self)

    def get_random_loot(self, reaction):
        roll = random.random()
        is_perfect = reaction < (self.time_limit / 2)
        if is_perfect and roll < 0.40:
            return self.loot_pool[-1]
        elif roll < 0.60:
            index = len(self.loot_pool) // 2
            return self.loot_pool[index]
        else:
            return self.loot_pool[0]


class FishBiomeView(View):
    def __init__(self, ctx):
        super().__init__(timeout=30)
        self.ctx = ctx

    async def launch_biome(self, interaction, biome, limit, items):
        if interaction.user != self.ctx.author:
            return

        embed = discord.Embed(
            title=f"🎣 Direction : {biome}",
            description="Ligne lancée... **Attends le signal VERT !**",
            color=discord.Color.blue()
        )
        game_view = FishingGameView(self.ctx, biome, limit, items)
        await interaction.response.edit_message(embed=embed, view=game_view)
        asyncio.create_task(game_view.start_game(interaction.message))

    @discord.ui.button(label="🦆 Étang (Facile)", style=discord.ButtonStyle.success)
    async def pond(self, interaction: discord.Interaction, button: Button):
        items = [OldBoot(), Trout(), Salmon()]
        await self.launch_biome(interaction, "L'Étang Paisible", 2.0, items)

    @discord.ui.button(label="🐟 Rivière (Moyen)", style=discord.ButtonStyle.primary)
    async def river(self, interaction: discord.Interaction, button: Button):
        items = [Salmon(), Sardine(), Carp(), Pufferfish()]
        await self.launch_biome(interaction, "La Rivière Agitée", 1.2, items)

    @discord.ui.button(label="🦈 Océan (Extrême)", style=discord.ButtonStyle.danger)
    async def ocean(self, interaction: discord.Interaction, button: Button):
        items = [Pufferfish(), Swordfish(), Shark(), Whale(), KrakenTentacle()]
        await self.launch_biome(interaction, "La Haute Mer", 0.7, items)


class Fishing(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='fish')
    @daily_limit("fish", 5)
    async def fish(self, ctx):
        embed = discord.Embed(
            title="🎣 Partie de Pêche",
            description="Où veux-tu aller pêcher aujourd'hui ?",
            color=discord.Color.teal()
        )
        embed.add_field(name="🦆 Étang", value="Facile. Pas de stress.", inline=True)
        embed.add_field(name="🐟 Rivière", value="Moyen. Courant modéré.", inline=True)
        embed.add_field(name="🦈 Océan", value="Extrême ! Pour les pêcheurs aguerris.", inline=True)
        lvl, _ = get_job_data(ctx.author.id, "fisher")
        embed.set_footer(text=f"Le délai augmente de 0.1s par niveau (tu es niveau {lvl})")
        view = FishBiomeView(ctx)
        await ctx.send(embed=embed, view=view)


async def setup(bot):
    await bot.add_cog(Fishing(bot))