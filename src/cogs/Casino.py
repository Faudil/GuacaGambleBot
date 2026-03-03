import asyncio
import random

import discord
from discord.ext import commands
from discord.ui import Button, View

from src.command_decorators import daily_limit, opening_hours, ActivityType
from src.database.balance import update_balance, get_balance
from src.database.item import has_item, remove_item_from_inventory
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.job import add_job_xp
from src.items.CheatCoin import CheatCoin

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


class CheatView(View):
    def __init__(self, author):
        super().__init__(timeout=30)
        self.author = author
        self.use_cheat = False

    @discord.ui.button(label="🕵️ Utiliser la Pièce Truquée (75%)", style=discord.ButtonStyle.danger)
    async def cheat(self, interaction: discord.Interaction, button: Button):
        if interaction.user.id != self.author.id:
            return
        self.use_cheat = True
        await interaction.response.defer()
        self.stop()

    @discord.ui.button(label="🎲 Jouer loyal (50%)", style=discord.ButtonStyle.secondary)
    async def legit(self, interaction: discord.Interaction, button: Button):
        if interaction.user.id != self.author.id:
            return
        self.use_cheat = False
        await interaction.response.defer()
        self.stop()


class Casino(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='coinflip', aliases=['cf', 'pileouface'])
    @daily_limit("coinflip", 10)
    async def coinflip(self, ctx, choice: str, amount: int):
        """Jouer à pile ou face contre le bot."""
        user_id = ctx.author.id
        choice = choice.lower()
        if choice not in ['pile', 'face', 'heads', 'tails']:
            return await ctx.send("❌ Choisis **pile** ou **face**.")

        if choice == 'heads':
            choice = 'face'
        if choice == 'tails':
            choice = 'pile'

        if amount <= 0:
            return await ctx.send("❌ Mise invalide.")
        if amount > 2000:
            return await ctx.send("❌ Tu ne peux pas miser plus de 2000$.")
        if get_balance(user_id) < amount:
            return await ctx.send("❌ Pas assez d'argent.")
        use_rigged = False
        if has_item(user_id, CheatCoin().name):
            view = CheatView(ctx.author)
            msg = await ctx.send(
                f"💳 Mise: **${amount}** sur **{choice.upper()}**.\n🕵️ Tu as une **Pièce Truquée** dans ta poche...",
                view=view)
            await view.wait()
            if view.use_cheat:
                if remove_item_from_inventory(user_id, CheatCoin().name):
                    use_rigged = True
                    await msg.edit(content="🕵️ *Tu échanges discrètement la pièce...*", view=None)
                else:
                    await ctx.send("❌ Erreur : Item introuvable au moment de l'utilisation.")
            else:
                await msg.edit(content="🎲 Tu décides de jouer à la loyal.", view=None)
        else:
            await ctx.send(f"🎲 C'est parti ! **{choice.upper()}** pour **${amount}**...")
            
        increment_stat(user_id, "coinflip_spent", amount)
        await asyncio.sleep(1)
        if use_rigged:
            if random.random() < 0.75:
                result_side = choice
                win = True
            else:
                result_side = "face" if choice == "pile" else "pile"
                win = False
        else:
            options = ["pile", "face"]
            result_side = random.choice(options)
            win = (result_side == choice)
        if win:
            xp_gain = 10
            update_balance(user_id, amount)
            increment_stat(user_id, "coinflip_won")
            increment_stat(user_id, "coinflip_money_won", amount)
            text = f"✨ **GAGNÉ !** La pièce tombe sur **{result_side.upper()}**."
            if use_rigged: text += " (Merci la triche 😉)"
            color = discord.Color.green()
        else:
            xp_gain = 30
            update_balance(user_id, -amount)
            increment_stat(user_id, "coinflip_lost")
            increment_stat(user_id, "coinflip_money_lost", amount)
            text = f"❌ **PERDU...** La pièce tombe sur **{result_side.upper()}**."
            if use_rigged: text += " (Même en trichant ?! La honte...)"
            color = discord.Color.red()
        add_job_xp(user_id, "gambler", xp_gain)
        text += f" Tu as gagné {xp_gain} xp"
        embed = discord.Embed(description=text, color=color)
        await ctx.send(embed=embed)
        
        unlocks = check_and_unlock_achievements(user_id)
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks))
        return


    @commands.command(name='slots', aliases=['slot', 'casino'])
    @daily_limit("slots", 10)
    async def slots(self, ctx, amount: int):
        """Jouer à la machine à sous."""
        user_id = ctx.author.id
        if amount <= 0:
            return await ctx.send("❌ Mise invalide.")
        bal = get_balance(user_id)
        if bal < amount:
            return await ctx.send(f"❌ Pas assez d'argent (${bal}).")
        update_balance(user_id, -amount)
        increment_stat(user_id, "slots_spent", amount)
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
            xp_gain = 100
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
            xp_gain = 30
        else:
            win_type = "LOSE"
            color = discord.Color.dark_red()
            xp_gain = 10
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
            increment_stat(user_id, "slots_won")
            update_balance(user_id, payout)
            status = f"{flavor}\n💰 **Gain : ${payout}**"
            net_profit = payout - amount
            if net_profit > 0:
                increment_stat(user_id, "slots_money_won", net_profit)
            elif net_profit < 0:
                increment_stat(user_id, "slots_money_lost", -net_profit)
        else:
            increment_stat(user_id, "slots_lost")
            increment_stat(user_id, "slots_money_lost", amount)
            status = f"{flavor}\n💸 -${amount}"
        status += f" +{xp_gain} xp"
        await msg.edit(embed=make_embed(r1, r2, r3, status, color))
        
        unlocks = check_and_unlock_achievements(user_id)
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks))
        return


async def setup(bot):
    await bot.add_cog(Casino(bot))
