import discord
from discord.ext import commands
import random

from src.database.balance import get_balance, update_balance
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.settings import get_language
from src.utils.i18n import t


class Duel(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.pending_duels = {}

    @commands.command(name='duel')
    async def duel(self, ctx, opponent: discord.Member, amount: int):
        """Provoque quelqu'un en duel (50/50)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        challenger = ctx.author
        if opponent.bot:
            return await ctx.send(t("duel.no_bot", lang))
        if opponent.id == challenger.id:
            return await ctx.send(t("duel.self_duel", lang))
        if amount <= 0:
            return await ctx.send(t("duel.invalid_bet", lang))
        challenger_bal = get_balance(challenger.id)
        opponent_bal = get_balance(opponent.id)
        if challenger_bal < amount:
            return await ctx.send(t("duel.no_money_self", lang, balance=challenger_bal))
        if opponent_bal < amount:
            return await ctx.send(t("duel.no_money_opponent", lang, user=opponent.display_name, balance=opponent_bal))
        
        self.pending_duels[opponent.id] = (challenger.id, amount)
        embed = discord.Embed(title=t("duel.challenge_title", lang), color=discord.Color.red())
        embed.description = t("duel.challenge_desc", lang, challenger=challenger.mention, opponent=opponent.mention, amount=amount, user=opponent.display_name)
        return await ctx.send(embed=embed)

    @commands.command(name='accept')
    async def accept_duel(self, ctx):
        """Accepter un duel de Quitte ou Double."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        opponent = ctx.author
        if opponent.id not in self.pending_duels:
            return await ctx.send(t("duel.no_challenge", lang))
        challenger_id, amount = self.pending_duels.pop(opponent.id)
        challenger = ctx.guild.get_member(challenger_id)
        if get_balance(challenger_id) < amount or get_balance(opponent.id) < amount:
            return await ctx.send(t("duel.money_spent_cancel", lang))
        
        die_1_challenger = random.randint(1, 6)
        die_2_challenger = random.randint(1, 6)
        total_challenger = die_1_challenger + die_2_challenger
        die_1_opponent = random.randint(1, 6)
        die_2_opponent = random.randint(1, 6)
        total_opponent = die_1_opponent + die_2_opponent
        
        result_msg = t("duel.roll_msg", lang, user=challenger.display_name, die1=die_1_challenger, die2=die_2_challenger, total=total_challenger)
        result_msg += t("duel.roll_msg", lang, user=opponent.display_name, die1=die_1_opponent, die2=die_2_opponent, total=total_opponent) + "\n"
        
        if total_challenger > total_opponent:
            update_balance(challenger_id, amount)
            update_balance(opponent.id, -amount)
            increment_stat(challenger_id, "pvp_wins")
            increment_stat(opponent.id, "pvp_losses")
            result_msg += t("duel.win_msg", lang, user=challenger.display_name, amount=amount)
            color = discord.Color.gold()
        elif total_opponent > total_challenger:
            update_balance(opponent.id, amount)
            update_balance(challenger_id, -amount)
            increment_stat(opponent.id, "pvp_wins")
            increment_stat(challenger_id, "pvp_losses")
            result_msg += t("duel.win_msg", lang, user=opponent.display_name, amount=amount)
            color = discord.Color.gold()
        else:
            result_msg += t("duel.draw_msg", lang)
            color = discord.Color.light_grey()
            
        embed = discord.Embed(title=t("duel.result_title", lang), description=result_msg, color=color)
        await ctx.send(embed=embed)
        
        c_unlocks = check_and_unlock_achievements(challenger_id)
        if c_unlocks:
             await ctx.send(content=challenger.mention, embed=format_achievements_unlocks(c_unlocks, lang))
        o_unlocks = check_and_unlock_achievements(opponent.id)
        if o_unlocks:
             await ctx.send(content=opponent.mention, embed=format_achievements_unlocks(o_unlocks, lang))
        return None

    @commands.command(name='deny')
    async def deny_duel(self, ctx):
        """Refuser un duel."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if ctx.author.id in self.pending_duels:
            del self.pending_duels[ctx.author.id]
            await ctx.send(t("duel.deny_msg", lang, user=ctx.author.display_name))
        else:
            await ctx.send(t("duel.no_deny", lang))


async def setup(bot):
    await bot.add_cog(Duel(bot))
