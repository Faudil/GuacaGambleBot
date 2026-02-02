import random

import discord
from discord.ext import commands, tasks

from src.command_decorators import daily_limit
from src.data_handling import get_balance, update_balance, pay_random_broke_user, get_bank_data
from src.globals import DAILY_AMOUNT, CHANNEL_ID


class Economy(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.random_welfare.start()

    def cog_unload(self):
        self.random_welfare.cancel()

    @tasks.loop(minutes=30)
    async def random_welfare(self):
        if random.random() < 0.5:
            amount = 30
            winner_id = pay_random_broke_user(amount, max_balance=50)
            if winner_id:
                for guild in self.bot.guilds:
                    member = guild.get_member(winner_id)
                    if member:
                        channel = guild.get_channel(CHANNEL_ID)
                        if not channel and guild.text_channels:
                            channel = guild.text_channels[0]
                        if channel:
                            embed = discord.Embed(
                                title="🍀 LE RSA 🍀",
                                description=f"Le Guacamolistan vient en aide à un joueur fauché !\n**{member.mention}** vient de recevoir **${amount}** du glorieux régime.",
                                color=discord.Color.green()
                            )
                            embed.set_thumbnail(url=member.display_avatar.url)
                            await channel.send(embed=embed)
                            break

    @random_welfare.before_loop
    async def before_welfare(self):
        await self.bot.wait_until_ready()

    @commands.command(name='balance', aliases=['bal'])
    async def balance(self, ctx, member: discord.Member = None):
        user = ctx.author if member is None else member
        wallet, bank = get_bank_data(user.id)
        interest = (bank // 100) * 10
        embed = discord.Embed(title="🏦 Ma Banque", color=discord.Color.blue())
        embed.add_field(name="👛 Portefeuille", value=f"${wallet}", inline=True)
        embed.add_field(name="🔒 Coffre-fort", value=f"${bank} / 500", inline=True)
        embed.add_field(name="📈 Intérêts Daily", value=f"+${interest} / jour", inline=False)
        embed.set_footer(text="Commandes : !dep <montant>, !withdraw <montant>")
        await ctx.send(embed=embed)

    @commands.command(name='daily')
    @daily_limit("daily", 1)
    async def daily(self, ctx):
        user_id = str(ctx.author.id)
        _, bank_bal = get_bank_data(user_id)
        interest = (bank_bal // 100) * 10
        new_balance = update_balance(user_id, DAILY_AMOUNT + interest)
        embed = discord.Embed(title="💸 Voilà ta thune", color=discord.Color.green())
        embed.add_field(name="Quantité", value=f"+${DAILY_AMOUNT}")
        embed.add_field(name="Ta balance", value=f"${new_balance}")
        embed.set_footer(text="Reviens demain !")
        return await ctx.send(embed=embed)

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
        embed = discord.Embed(title="💸 Transaction complète", color=discord.Color.green())
        embed.add_field(name="Donneur", value=ctx.author.display_name, inline=True)
        embed.add_field(name="Receveur", value=recipient.display_name, inline=True)
        embed.add_field(name="Quantité", value=f"**${amount}**", inline=False)
        return await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Economy(bot))
