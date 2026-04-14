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
from src.database.settings import get_language
from src.utils.i18n import t


class MineExpeditionView(View):
    def __init__(self, ctx, risk_reduc, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.risk_reduc = risk_reduc
        self.lang = lang
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

    @discord.ui.button(label="dig_placeholder", style=discord.ButtonStyle.primary)
    async def dig(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return

        risk = (self.depth - 1) * 5
        risk -= self.risk_reduc

        roll = random.randint(1, 100)
        if roll <= risk:
            self.is_collapsed = True
            lost_items = ", ".join([loot.display_name(self.lang) for loot in self.loot_bag]) if self.loot_bag else t("mining.nothing", self.lang)
            msg = t("mining.collapse_msg", self.lang, items=lost_items)
            await self.update_message(interaction, msg, end=True)
            return

        drop = self.get_loot()
        self.loot_bag.append(drop)
        self.depth += 1

        bag_str = ", ".join([loot.display_name(self.lang) for loot in self.loot_bag])
        risk_next = max(0, ((self.depth - 1) * 5) - self.risk_reduc)
        msg = t("mining.status", self.lang, depth=self.depth, item=drop.display_name(self.lang), bag=bag_str, risk=risk_next)
        await self.update_message(interaction, msg)

    @discord.ui.button(label="leave_placeholder", style=discord.ButtonStyle.success)
    async def leave(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        total_xp = self.depth * 5
        if self.loot_bag:
            for item in self.loot_bag:
                add_item_to_inventory(self.ctx.author.id, item.name)
                increment_stat(self.ctx.author.id, "items_mined")
                total_xp += 10
            bag_str = ", ".join([loot.display_name(self.lang) for loot in self.loot_bag])
            msg = t("mining.success_msg", self.lang, bag=bag_str, xp=total_xp)
        else:
            msg = t("mining.empty_msg", self.lang, xp=total_xp)
        add_job_xp(self.ctx.author.id, "miner", total_xp)
        await self.update_message(interaction, msg, end=True)
        self.stop()
        
        unlocks = check_and_unlock_achievements(self.ctx.author.id)
        if unlocks:
            await interaction.channel.send(content=interaction.user.mention, embed=format_achievements_unlocks(unlocks, self.lang))


class Mine(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name="mine", aliases=["mining", "miner"])
    @daily_limit("mine", 10)
    async def mine(self, ctx):
        """Expédition minière. Gère ton risque pour trouver des diamants."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = int(ctx.message.author.id)
        
        from src.database.housing import is_inventory_full
        full, current, limit = is_inventory_full(user_id)
        if full:
            return await ctx.send(t("housing.inv_full_warning", lang, current=current, limit=limit))
            
        lvl, _ = get_job_data(user_id, "miner")
        embed = discord.Embed(title=t("mining.title", lang), description=t("mining.desc", lang))
        pet = get_active_pet(user_id)
        risk_reduc = lvl
        if pet is not None and pet.bonus == PetBonus.MINE:
            risk_reduc += pet.level // 4
        embed.set_footer(text=t("mining.footer", lang, risk=risk_reduc))

        view = MineExpeditionView(ctx, risk_reduc, lang)
        # Update button labels
        for item in view.children:
            if isinstance(item, Button):
                 if item.label == "dig_placeholder": item.label = t("mining.dig_label", lang)
                 if item.label == "leave_placeholder": item.label = t("mining.leave_label", lang)

        await ctx.send(embed=embed, view=view)
        try:
            await ctx.message.delete()
        except:
            pass


async def setup(bot):
    await bot.add_cog(Mine(bot))