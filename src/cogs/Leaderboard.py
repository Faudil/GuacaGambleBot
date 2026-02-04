import discord
from discord.ext import commands

from src.database.other import get_top_users


class Leaderboard(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
    @commands.command(name='top', aliases=['leaderboard', 'classement'])
    async def leaderboard(self, ctx):
        """Display the 5 richest in the server."""
        users = list(get_top_users(5))
        if not users:
            return await ctx.send("Personne n'a d'argent pour l'instant !")
        sorted_users = sorted(users, key=lambda x: x["balance"], reverse=True)
        embed = discord.Embed(title="🏆 Classement des plus riches", color=discord.Color.gold())
        description = ""
        for i, user in enumerate(sorted_users, 1):
            member = ctx.guild.get_member(int(user["user_id"]))
            name = member.display_name if member else "Inconnu"
            medals = {1: "🥇", 2: "🥈", 3: "🥉"}
            rank_emoji = medals.get(i, "🔹")
            description += f"{rank_emoji} **{name}** : ${user['balance']}\n"
        embed.description = description
        return await ctx.send(embed=embed)

async def setup(bot):
    await bot.add_cog(Leaderboard(bot))
