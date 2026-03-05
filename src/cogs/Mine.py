from typing import List

from src.database.item import add_item_to_inventory
from src.database.job import get_job_data, add_job_xp
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks

import discord
from discord.ext import commands
from discord.ui import View, Button
import random
from src.command_decorators import daily_limit, opening_hours, ActivityType
from src.database.pets import get_active_pet

from src.items.MiningLoot import Pebble, Diamond, IronOre, GoldNugget, ResourceItem, Coal, CopperOre, SilverOre, \
    PlatinumOre, Emerald
from src.models.Pet import PetBonus

XP_PER_RARITY = {

}

class MineExpeditionView(View):
    def __init__(self, ctx, risk_reduc):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.risk_reduc = risk_reduc
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
        risk -= self.risk_reduc

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
            f"Tu as trouvé : **{drop.name}** !\n\n"
            f"🎒 **Sac actuel :** {bag_str}\n"
            f"⚠️ **Risque d'éffondrement au prochain coup :** ~{max(0, ((self.depth - 1) * 5) - self.risk_reduc)}%"
        )
        await self.update_message(interaction, msg)

    @discord.ui.button(label="🏃 Sortir et Sécuriser", style=discord.ButtonStyle.success)
    async def leave(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        total_xp = self.depth * 5
        if self.loot_bag:
            for item in self.loot_bag:
                add_item_to_inventory(self.ctx.author.id, item.name)
                increment_stat(self.ctx.author.id, "items_mined")
                total_xp += 10
            bag_str = ", ".join([loot.name for loot in self.loot_bag])
            msg = f"✅ **Mission Réussie !** Tu sors de la mine vivant.\n🎒 Tu remportes : {bag_str}\n📈 XP gagnée : +{total_xp}"
        else:
            msg = f"😐 Tu sors de la mine les mains vides. Au moins tu as acquis de l'expérience\n📈 XP gagnée : +{total_xp}"
        add_job_xp(self.ctx.author.id, "miner", total_xp)
        await self.update_message(interaction, msg, end=True)
        self.stop()
        
        unlocks = check_and_unlock_achievements(self.ctx.author.id)
        if unlocks:
            await interaction.channel.send(content=interaction.user.mention, embed=format_achievements_unlocks(unlocks))


class Mine(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name="mine", aliases=["mining", "miner"])
    @daily_limit("mine", 10)
    async def mine(self, ctx):
        """Expédition minière. Gère ton risque pour trouver des diamants."""
        user_id = int(ctx.message.author.id)
        lvl, _ = get_job_data(user_id, "miner")
        embed = discord.Embed(title="Expédition minière", description="Tu entres dans la grotte...\nJusqu'où iras-tu ?")
        pet = get_active_pet(user_id)
        risk_reduc = lvl
        if pet is not None and pet.bonus == PetBonus.MINE:
            risk_reduc += pet.level // 4
        embed.set_footer(text=f"Risques réduit de : {risk_reduc} % (Bonus niveau et animal de compagnie)")

        view = MineExpeditionView(ctx, risk_reduc)
        await ctx.send(embed=embed, view=view)
        await ctx.message.delete()


async def setup(bot):
    await bot.add_cog(Mine(bot))