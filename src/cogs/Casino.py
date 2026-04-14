from discord.ext import commands
from discord.ui import Button, View
import asyncio
import random
import discord

from src.command_decorators import daily_limit
from src.database.balance import update_balance, get_balance
from src.database.item import has_item, remove_item_from_inventory
from src.database.achievement import check_and_unlock_achievements, format_achievements_unlocks
from src.database.job import add_job_xp
from src.database.settings import get_language
from src.database.achievement import increment_stat
from src.items.CheatCoin import CheatCoin
from src.utils.i18n import t

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


def get_flavor_text(win_type, symbol, lang):
    if win_type == "JACKPOT":
        if symbol == "💎": return t("slots.jackpot_diamond", lang)
        if symbol == "🔔": return t("slots.jackpot_bell", lang)
        return t("slots.jackpot_generic", lang, symbol=symbol)
    if win_type == "PAIRE":
        if symbol == "💎": return t("slots.pair_diamond", lang)
        if symbol == "🔔": return t("slots.pair_bell", lang)
        if symbol == "🍋": return t("slots.pair_lemon", lang)
        if symbol == "🍇": return t("slots.pair_grape", lang)
        if symbol == "🍒": return t("slots.pair_cherry", lang)
    return t("slots.lose_generic", lang)


class CheatView(View):
    def __init__(self, author, lang):
        super().__init__(timeout=30)
        self.author = author
        self.lang = lang
        self.use_cheat = False

    @discord.ui.button(label="cheat", style=discord.ButtonStyle.danger)
    async def cheat(self, interaction: discord.Interaction, button: Button):
        if interaction.user.id != self.author.id:
            return
        self.use_cheat = True
        await interaction.response.defer()
        self.stop()

    @discord.ui.button(label="legit", style=discord.ButtonStyle.secondary)
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
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = ctx.author.id
        choice = choice.lower()
        if choice not in ['pile', 'face', 'heads', 'tails']:
            return await ctx.send(t("coinflip.choice_error", lang))

        if choice == 'heads':
            choice = 'face'
        if choice == 'tails':
            choice = 'pile'

        if amount <= 0:
            return await ctx.send(t("coinflip.invalid_bet", lang))
        if amount > 2000:
            return await ctx.send(t("coinflip.max_bet", lang))
        if get_balance(user_id) < amount:
            return await ctx.send(t("coinflip.no_money", lang))
            
        use_rigged = False
        if has_item(user_id, CheatCoin().name):
            view = CheatView(ctx.author, lang)
            for item in view.children:
                if isinstance(item, Button):
                    if item.label == "cheat": item.label = t("coinflip.cheat_label", lang)
                    if item.label == "legit": item.label = t("coinflip.legit_label", lang)

            msg = await ctx.send(
                t("coinflip.cheat_info", lang, amount=amount, choice=choice.upper()),
                view=view)
            await view.wait()
            if view.use_cheat:
                if remove_item_from_inventory(user_id, CheatCoin().name):
                    use_rigged = True
                    await msg.edit(content=t("coinflip.cheat_swap", lang), view=None)
                else:
                    await ctx.send("❌ Error: Item not found.")
            else:
                await msg.edit(content=t("coinflip.legit_choice", lang), view=None)
        else:
            await ctx.send(t("coinflip.start_msg", lang, choice=choice.upper(), amount=amount))
            
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
            text = t("coinflip.win_msg", lang, result=result_side.upper())
            if use_rigged: text += t("coinflip.win_cheat", lang)
            color = discord.Color.green()
        else:
            xp_gain = 30
            update_balance(user_id, -amount)
            increment_stat(user_id, "coinflip_lost")
            increment_stat(user_id, "coinflip_money_lost", amount)
            text = t("coinflip.lose_msg", lang, result=result_side.upper())
            if use_rigged: text += t("coinflip.lose_cheat", lang)
            color = discord.Color.red()
            
        add_job_xp(user_id, "gambler", xp_gain)
        from src.database.npc import add_reputation
        rep_added = add_reputation(user_id, "gamblebot", 5)
        text += t("coinflip.xp_gain", lang, xp=xp_gain)
        if rep_added > 0:
            text += f"\n🤖 +{rep_added} Rép. GambleBot"
        embed = discord.Embed(description=text, color=color)
        await ctx.send(embed=embed)
        
        unlocks = check_and_unlock_achievements(user_id)
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks, lang))
        return


    @commands.command(name='slots', aliases=['slot', 'casino'])
    @daily_limit("slots", 10)
    async def slots(self, ctx, amount: int):
        """Jouer à la machine à sous."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = ctx.author.id
        if amount <= 0:
            return await ctx.send(t("coinflip.invalid_bet", lang))
        bal = get_balance(user_id)
        if bal < amount:
            return await ctx.send(t("slots.no_money", lang, balance=bal))
            
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
            
        flavor = get_flavor_text(win_type, winning_symbol, lang)
        
        def make_embed(s1, s2, s3, state_text, col):
            emb = discord.Embed(title=t("slots.title", lang), color=col)
            machine_display = f"**»** {s1}   |   {s2}   |   {s3}  ****"
            emb.add_field(name="Machine", value=f"# {machine_display}", inline=False)
            emb.add_field(name="Infos", value=f"{t('blackjack.pot', lang, amount=amount).replace('Pot', t('economy.quantity', lang))}\n{state_text}", inline=False)
            return emb
            
        rolling_sym = t("slots.rolling", lang)
        msg = await ctx.send(embed=make_embed(rolling_sym, rolling_sym, rolling_sym, t("slots.state_start", lang), discord.Color.blurple()))
        await asyncio.sleep(1.0)
        await msg.edit(embed=make_embed(r1, rolling_sym, rolling_sym, t("slots.state_rolling", lang), discord.Color.blurple()))
        await asyncio.sleep(0.5)
        await msg.edit(embed=make_embed(r1, r2, rolling_sym, t("slots.state_suspense", lang), discord.Color.blurple()))
        if r1 == r2:
            await asyncio.sleep(1.5)
        else:
            await asyncio.sleep(0.5)
            
        if is_win:
            increment_stat(user_id, "slots_won")
            update_balance(user_id, payout)
            status = f"{flavor}\n{t('slots.gain', lang, amount=payout)}"
            net_profit = payout - amount
            if net_profit > 0:
                increment_stat(user_id, "slots_money_won", net_profit)
            elif net_profit < 0:
                increment_stat(user_id, "slots_money_lost", -net_profit)
        else:
            increment_stat(user_id, "slots_lost")
            increment_stat(user_id, "slots_money_lost", amount)
            status = f"{flavor}\n{t('slots.loss', lang, amount=amount)}"
            
        status += t("coinflip.xp_gain", lang, xp=xp_gain)
        from src.database.npc import add_reputation
        rep_added = add_reputation(user_id, "gamblebot", 5)
        if rep_added > 0:
            status += f"\n🤖 +{rep_added} Rép. GambleBot"
        await msg.edit(embed=make_embed(r1, r2, r3, status, color))
        
        unlocks = check_and_unlock_achievements(user_id)
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks))
        return


async def setup(bot):
    await bot.add_cog(Casino(bot))
