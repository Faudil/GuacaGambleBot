import random

import discord
from discord.ext import commands

from src.command_decorators import daily_limit
from src.data_handling import get_balance, update_balance


class Casino(commands.Cog):
    def __init__(self, bot):
        self.bot = bot


    @commands.command(name='coinflip', aliases=['cf', 'pileouface'])
    @daily_limit("coinflip", 10)
    async def coinflip(self, ctx, choice: str, amount: int):
        """Bet against the bot. Usage: !cf <pile/face> <amount>"""
        user_id = str(ctx.author.id)
        choice = choice.lower()
        valid_choices = ["pile", "face"]
        if choice not in valid_choices:
            return await ctx.send("❌ Choisis **pile** ou **face**.")
        if amount <= 0:
            return await ctx.send("❌ Tu dois parier plus que 0$.")
        user_bal = get_balance(user_id)
        if user_bal < amount:
            return await ctx.send(f"❌ Tu n'as pas assez d'argent (${user_bal}).")
        outcome = random.choice(valid_choices)
        if choice == outcome:
            update_balance(user_id, amount)
            return await ctx.send(f"🪙 La pièce tombe sur **{outcome.upper()}** ! Tu gagnes **${amount}** ! Tu possèdes maintenant {user_bal + amount}$. 🎉")
        else:
            update_balance(user_id, -amount)
            return await ctx.send(f"🪙 La pièce tombe sur **{outcome.upper()}**... Tu perds **${amount}**. Tu possèdes maintenant {user_bal - amount}$. 😢")


    @commands.command(name='slots', aliases=['slot', 'casino'])
    @daily_limit("slots", 5)
    async def slots(self, ctx, amount: int):
        """Joue à la machine à sous. Usage: !slots <montant>"""
        user_id = str(ctx.author.id)
        bal = get_balance(user_id)
        if amount <= 0:
            return await ctx.send("❌ Mise invalide.")
        if bal < amount:
            return await ctx.send("❌ Pas assez d'argent.")
        update_balance(user_id, -amount)
        symbols = ["🍒", "🍋", "🍇", "🍉", "7️⃣", "💎"]
        s1 = random.choice(symbols)
        s2 = random.choice(symbols)
        s3 = random.choice(symbols)
        winnings = 0
        result_text = "Perdu..."
        if s1 == s2 == s3:
            winnings = amount * 10
            result_text = "🚨 **JACKPOT !** 🚨 (x10)"
        elif s1 == s2 or s2 == s3 or s1 == s3:
            winnings = amount * 2
            result_text = "Pas mal ! Deux identiques. (x2)"
        if winnings > 0:
            update_balance(user_id, winnings)
        embed = discord.Embed(title="🎰 Machine à Sous", color=discord.Color.magenta())
        embed.description = f"# ║ {s1} ║ {s2} ║ {s3} ║"
        embed.add_field(name="Résultat", value=result_text)
        embed.add_field(name="Gain", value=f"+${winnings}" if winnings > 0 else f"-${amount}")
        return await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Casino(bot))
