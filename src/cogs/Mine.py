from typing import List

from src.database.item import add_item_to_inventory
from src.database.job import get_job_data, add_job_xp

import discord
from discord.ext import commands
from discord.ui import View, Button
import random
from src.command_decorators import daily_limit

from src.items.MiningLoot import Pebble, Diamond, IronOre, GoldNugget, ResourceItem, Coal, CopperOre, SilverOre, \
    PlatinumOre, Emerald


XP_PER_RARITY = {

}

class MineExpeditionView(View):
    def __init__(self, ctx, initial_level):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.level = initial_level
        self.depth = 1
        self.loot_bag: List[ResourceItem] = []
        self.is_collapsed = False

    def get_loot(self) -> ResourceItem:
        roll = random.random()

        if self.depth == 1:
            return Pebble() if roll < 0.70 else Coal()
        elif self.depth <= 4:
            if roll < 0.40:
                return Coal()
            elif roll < 0.75:
                return IronOre()
            else:
                return CopperOre()
        elif self.depth <= 7:
            if roll < 0.40:
                return CopperOre()
            elif roll < 0.80:
                return SilverOre()
            else:
                return GoldNugget()
        else:
            if roll < 0.40:
                return GoldNugget()
            elif roll < 0.70:
                return PlatinumOre()
            elif roll < 0.90:
                return Emerald()
            else:
                return Diamond()


    async def update_message(self, interaction, content, end=False):
        if end:
            self.clear_items()
        await interaction.response.edit_message(content=content, view=self)

    @discord.ui.button(label="⛏️ Creuser plus profond", style=discord.ButtonStyle.primary)
    async def dig(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return

        risk = (self.depth - 1) * 5
        risk -= self.level
        risk = max(5, risk)

        roll = random.randint(1, 100)
        if roll <= risk:
            self.is_collapsed = True
            msg = (
                f"💥 **CATASTROPHE !** La galerie s'effondre sur toi !\n"
                f"Tu lâches ton sac pour t'enfuir en courant.\n"
                f"❌ **Perdu :** {", ".join([loot.name for loot in self.loot_bag]) if self.loot_bag else 'Rien (ouf)'}"
            )
            await self.update_message(interaction, msg, end=True)
            return

        drop = self.get_loot()
        self.loot_bag.append(drop)
        self.depth += 1

        bag_str = ", ".join([loot.name for loot in self.loot_bag])
        msg = (
            f"⛏️ **Profondeur {self.depth}m**\n"
            f"Tu as trouvé : **{drop}** !\n\n"
            f"🎒 **Sac actuel :** {bag_str}\n"
            f"⚠️ **Risque d'éffondrement au prochain coup :** ~{max(0, (self.depth * 5) - self.level)}%"
        )
        await self.update_message(interaction, msg)

    @discord.ui.button(label="🏃 Sortir et Sécuriser", style=discord.ButtonStyle.success)
    async def leave(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        total_xp = self.depth * 5
        if self.loot_bag:
            for item in self.loot_bag:
                add_item_to_inventory(self.ctx.author.id, item)
                total_xp += 10
            bag_str = ", ".join([loot.name for loot in self.loot_bag])
            msg = f"✅ **Mission Réussie !** Tu sors de la mine vivant.\n🎒 Tu remportes : {bag_str}\n📈 XP gagnée : +{total_xp}"
        else:
            msg = f"😐 Tu sors de la mine les mains vides. Au moins tu as acquis de l'expérience\n📈 XP gagnée : +{total_xp}"
        add_job_xp(self.ctx.author.id, "miner", total_xp)
        await self.update_message(interaction, msg, end=True)
        self.stop()


class Mine(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name="mine", aliases=["mining", "miner"])
    @daily_limit("mine", 2)
    async def mine(self, ctx):
        user_id = int(ctx.message.author.id)
        lvl, _ = get_job_data(user_id, "mining")
        embed = discord.Embed(title="Expédition minière", description="Tu entres dans la grotte...\nJusqu'où iras-tu ?")
        embed.set_footer(text=f"Niveau Mineur : {lvl} (Réduit les risques)")
        await ctx.message.delete()


async def setup(bot):
    await bot.add_cog(Mine(bot))