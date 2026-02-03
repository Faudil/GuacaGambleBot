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
for sym, data in SLOT_SYMBOLS.items():
    WHEEL.extend([sym] * data['weight'])


def get_flavor_text(win_type, symbol):
    """Returns a message based on gain."""
    if win_type == "JACKPOT":
        if symbol == "💎": return "💎 **OMEGA JACKPOT !!!** TU ES RICHE !!! 💎"
        if symbol == "🔔": return "🔔 **DING DING DING !** Le gros lot est pour toi !"
        return f"🎉 **INCROYABLE !** Trois {symbol} alignés !"
    if win_type == "PAIRE":
        if symbol == "💎": return "😲 **Si proche !** Une paire de diamants, ça rapporte gros !"
        if symbol == "🔔": return "🔔 **Joli !** Ces cloches sonnent la victoire."
        if symbol == "🍋": return "🍋 **Pas mal !** Un petit jus de citron pour fêter ça ?"
        if symbol == "🍇": return "🍇 **Ouf !** Au moins t'es remboursé (ou presque)."
        if symbol == "🍒": return "🍒 **Mince !** Ca aurait pu être pire."
    return "❌ Pas de chance... Retente ta chance !"


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
        user_id = ctx.author.id
        if amount <= 0:
            return await ctx.send("❌ Mise invalide.")
        bal = get_balance(user_id)
        if bal < amount:
            return await ctx.send(f"❌ Pas assez d'argent (${bal}).")
        update_balance(user_id, -amount)
        # increment_user_stat(user_id, "total_gambles", 1)

        r1 = random.choice(WHEEL)
        r2 = random.choice(WHEEL)
        r3 = random.choice(WHEEL)

        payout = 0
        is_win = False
        winning_symbol = None

        if r1 == r2 == r3:
            winning_symbol = r1
            multiplier = SLOT_SYMBOLS[winning_symbol]['mult']
            payout = amount * multiplier
            is_win = True
            win_type = "JACKPOT"
            color = discord.Color.gold()
        elif r1 == r2 or r2 == r3 or r1 == r3:
            winning_symbol = r1 if r1 == r2 else (r2 if r2 == r3 else r1)
            full_mult = SLOT_SYMBOLS[winning_symbol]['mult']
            ratio = 0.18
            payout = int(amount * full_mult * ratio)
            if payout < amount and winning_symbol in ["💎", "🔔", "🍋"]:
                payout = amount
            is_win = True
            win_type = "PAIRE"
            color = discord.Color.green() if payout > amount else discord.Color.blue()
        else:
            win_type = "LOSE"
            color = discord.Color.dark_red()
        flavor = get_flavor_text(win_type, winning_symbol)
        def make_embed(s1, s2, s3, state_text, col):
            emb = discord.Embed(title="🎰 CASINO", color=col)
            machine_display = f"**»** {s1}   |   {s2}   |   {s3}  ****"
            emb.add_field(name="Machine", value=f"# {machine_display}", inline=False)
            emb.add_field(name="Infos", value=f"Mise : **${amount}**\n{state_text}", inline=False)
            return emb
        msg = await ctx.send(embed=make_embed("🌀", "🌀", "🌀", "Faites vos jeux...", discord.Color.blurple()))
        await asyncio.sleep(1.0)
        await msg.edit(embed=make_embed(r1, "🌀", "🌀", "...", discord.Color.blurple()))
        await asyncio.sleep(0.5)
        await msg.edit(embed=make_embed(r1, r2, "🌀", "Suspense...", discord.Color.blurple()))
        if r1 == r2:
            await asyncio.sleep(1.5)
        else:
            await asyncio.sleep(0.5)
        if is_win:
            update_balance(user_id, payout)
            status = f"{flavor}\n💰 **Gain : ${payout}**"
        else:
            status = f"{flavor}\n💸 -${amount}"
        return await msg.edit(embed=make_embed(r1, r2, r3, status, color))


async def setup(bot):
    await bot.add_cog(Casino(bot))
