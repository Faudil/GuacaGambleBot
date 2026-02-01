import discord
from discord.ext import commands

from src.command_decorators import daily_limit
from src.data_handling import get_balance, update_balance
from src.globals import DAILY_AMOUNT


class Economy(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='balance', aliases=['bal'])
    async def balance(self, ctx, member: discord.Member = None):
        user = ctx.author if member is None else member
        bal = get_balance(user.id)
        embed = discord.Embed(title="💰 Compte bancaire", color=discord.Color.green())
        embed.add_field(name="Utilisateur", value=user.display_name)
        embed.add_field(name="Balance", value=f"${bal}")
        await ctx.send(embed=embed)


    @commands.command(name='daily')
    @daily_limit("daily", 1)
    async def daily(self, ctx):
        user_id = str(ctx.author.id)
        update_balance(user_id, DAILY_AMOUNT)
        await ctx.send(f"💸 You collected ${DAILY_AMOUNT}!")

    @commands.command(name='give', aliases=['pay'])
    async def give(self, ctx, recipient: discord.Member, amount: int) -> None:
        sender_id = ctx.author.id
        recipient_id = recipient.id
        if sender_id == recipient_id or amount <= 0:
            return await ctx.send("❌ Transaction invalide.")
        if get_balance(sender_id) < amount:
            return await ctx.send("❌ T'as pas assez d'argent.")
        update_balance(sender_id, -amount)
        update_balance(recipient_id, amount)
        return await ctx.send(f"💸 {ctx.author.display_name} a envoyé ${amount} vers {recipient.display_name}!")


async def setup(bot):
    await bot.add_cog(Economy(bot))
