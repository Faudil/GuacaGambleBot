import discord
from discord.ext import commands

from src.command_decorators import daily_limit
from src.data_handling import get_balance, update_balance
from src.globals import DAILY_AMOUNT


class Admin(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='airdrop', aliases=['rain'])
    @commands.has_permissions(administrator=True)
    async def airdrop(self, ctx, user: discord.Member, amount: int):
        """(Admin) Donne de l'argent à tout le monde !"""
        if amount <= 0:
            return await ctx.send("❌ Le montant doit être positif.")
        update_balance(user.id, amount)
        embed = discord.Embed(
            title="💸 PLUIE DE BILLETS ! 💸",
            description=f"Je donnne **${amount}** à mon préféré {user.display_name} !",
            color=discord.Color.gold()
        )
        embed.set_footer(text=f"Offert par {ctx.author.display_name}")
        return await ctx.send(embed=embed)

    @airdrop.error
    async def airdrop_error(self, ctx, error):
        if isinstance(error, commands.MissingPermissions):
            await ctx.send("⛔ Tu n'as pas la permission de faire pleuvoir l'argent !")


async def setup(bot):
    await bot.add_cog(Admin(bot))
