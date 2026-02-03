import discord
from discord.ext import commands
import random
import asyncio

# Attention aux imports : on utilise bien "src." maintenant
from src.data_handling import get_balance, update_balance


class Duel(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

        self.pending_duels = {}

    @commands.command(name='duel')
    async def duel(self, ctx, opponent: discord.Member, amount: int):
        challenger = ctx.author
        if opponent.bot:
            return await ctx.send("🤖 Tu ne peux pas défier un bot (ils trichent).")
        if opponent.id == challenger.id:
            return await ctx.send("❌ Tu ne peux pas te défier toi-même.")
        if amount <= 0:
            return await ctx.send("❌ La mise doit être positive.")
        challenger_bal = get_balance(challenger.id)
        opponent_bal = get_balance(opponent.id)
        if challenger_bal < amount:
            return await ctx.send(f"❌ Tu n'as pas assez d'argent (${challenger_bal}).")
        if opponent_bal < amount:
            return await ctx.send(f"❌ {opponent.display_name} n'a pas assez d'argent (${opponent_bal}).")
        self.pending_duels[opponent.id] = (challenger.id, amount)
        embed = discord.Embed(title="⚔️ DÉFI LANCÉ !", color=discord.Color.red())
        embed.description = (f"{challenger.mention} défie {opponent.mention} pour **${amount}** !\n"
                             f"**{opponent.display_name}**, tape `!accept` pour te battre ou `!deny` pour fuir.")
        return await ctx.send(embed=embed)

    @commands.command(name='accept')
    async def accept_duel(self, ctx):
        opponent = ctx.author
        if opponent.id not in self.pending_duels:
            return await ctx.send("❌ Personne ne t'a défié.")
        challenger_id, amount = self.pending_duels.pop(opponent.id)
        challenger = ctx.guild.get_member(challenger_id)
        if get_balance(challenger_id) < amount or get_balance(opponent.id) < amount:
            return await ctx.send("❌ L'un de vous n'a plus assez d'argent. Défi annulé.")
        die_1_challenger = random.randint(1, 6)
        die_2_challenger = random.randint(1, 6)
        total_challenger = die_1_challenger + die_2_challenger
        die_1_opponent = random.randint(1, 6)
        die_2_opponent = random.randint(1, 6)
        total_opponent = die_1_opponent + die_2_opponent
        result_msg = (
            f"🎲 **{challenger.display_name}** lance : {die_1_challenger} + {die_2_challenger} = **{total_challenger}**\n"
            f"🎲 **{opponent.display_name}** lance : {die_1_opponent} + {die_2_opponent} = **{total_opponent}**\n\n")
        if total_challenger > total_opponent:
            update_balance(challenger_id, amount)
            update_balance(opponent.id, -amount)
            result_msg += f"🏆 **{challenger.display_name}** gagne **${amount}** !"
            color = discord.Color.gold()
        elif total_opponent > total_challenger:
            update_balance(opponent.id, amount)
            update_balance(challenger_id, -amount)
            result_msg += f"🏆 **{opponent.display_name}** gagne **${amount}** !"
            color = discord.Color.gold()
        else:
            result_msg += "🤝 **Égalité !** Personne ne perd d'argent."
            color = discord.Color.light_grey()
        embed = discord.Embed(title="🎲 RÉSULTAT DU DUEL", description=result_msg, color=color)
        await ctx.send(embed=embed)
        return None

    @commands.command(name='deny')
    async def deny_duel(self, ctx):
        if ctx.author.id in self.pending_duels:
            del self.pending_duels[ctx.author.id]
            await ctx.send(f"🛡️ {ctx.author.display_name} a refusé le duel (le fragile !).")
        else:
            await ctx.send("❌ Aucun défi à refuser.")


async def setup(bot):
    await bot.add_cog(Duel(bot))
