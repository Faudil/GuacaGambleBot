import asyncio
import random

import discord
from discord.ext import commands

from src.command_decorators import daily_limit
from src.data_handling import get_balance, update_balance

SLOT_SYMBOLS = {
    "🍒": {"weight": 40, "mult": 3},
    "🍇": {"weight": 30, "mult": 5},
    "🍋": {"weight": 20, "mult": 10},
    "🔔": {"weight": 8,  "mult": 20},
    "💎": {"weight": 2,  "mult": 100}
}
WHEEL = []
for symbol, data in SLOT_SYMBOLS.items():
    WHEEL.extend([symbol] * data['weight'])


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
    @daily_limit("slots", 10)
    async def slots(self, ctx, amount: int):
        """Let's go gambling !"""
        user_id = ctx.author.id
        if amount <= 0:
            return await ctx.send("❌ La mise doit être positive.")
        bal = get_balance(user_id)
        if bal < amount:
            return await ctx.send(f"❌ Tu n'as pas assez d'argent (${bal}).")

        update_balance(user_id, -amount)
        # increment_user_stat(user_id, "total_gambles", 1)  # For later

        r1 = random.choice(WHEEL)
        r2 = random.choice(WHEEL)
        r3 = random.choice(WHEEL)

        if r1 == r2 == r3:
            multiplier = SLOT_SYMBOLS[r1]['mult']
            payout = amount * multiplier
            is_win = True
            win_type = "JACKPOT"
        elif r1 == r2 or r2 == r3 or r1 == r3:
            symbol = r1 if r1 == r2 else (r2 if r2 == r3 else r1)
            full_mult = SLOT_SYMBOLS[symbol]['mult']
            ratio = 0.25
            payout = int(amount * full_mult * ratio)
            is_win = True
            win_type = "PAIRE"
        else:
            payout = 0
            is_win = False
        embed = discord.Embed(title="🎰 CASINO SLOTS", color=discord.Color.blurple())
        embed.description = (
            f"Mise : **${amount}**\n\n"
            f"╔═════════════╗\n"
            f"║ 🌀 | 🌀 | 🌀 ║\n"
            f"╚═════════════╝"
        )
        msg = await ctx.send(embed=embed)
        await asyncio.sleep(1.0)
        embed.description = (
            f"Mise : **${amount}**\n\n"
            f"╔═════════════╗\n"
            f"║ {r1} | 🌀 | 🌀 ║\n"
            f"╚═════════════╝"
        )
        await msg.edit(embed=embed)
        await asyncio.sleep(0.8)
        embed.description = (
            f"Mise : **${amount}**\n\n"
            f"╔═════════════╗\n"
            f"║ {r1} | {r2} | 🌀 ║\n"
            f"╚═════════════╝"
        )
        await msg.edit(embed=embed)
        if r1 == r2:
            await asyncio.sleep(1.5)
        else:
            await asyncio.sleep(0.8)
        if is_win:
            update_balance(user_id, payout)
            embed.color = discord.Color.green()
            result_text = f"✨ **GAGNÉ !** ({win_type})\nTu remportes **${payout}** !"
        else:
            embed.color = discord.Color.red()
            result_text = f"❌ **PERDU...**\nTu perds tes ${amount}."

        embed.description = (
            f"Mise : **${amount}**\n\n"
            f"╔═════════════╗\n"
            f"║ {r1} | {r2} | {r3} ║\n"
            f"╚═════════════╝\n\n"
            f"{result_text}"
        )
        return await msg.edit(embed=embed)


async def setup(bot):
    await bot.add_cog(Casino(bot))
