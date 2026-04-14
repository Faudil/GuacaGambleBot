from discord.ext import commands
import discord

from src.database.balance import get_balance, update_balance
from src.database.bet import create_bet_db, get_bet_data, add_wager, close_bet_db, freeze_bet
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.settings import get_language
from src.utils.i18n import t


class Betting(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name='createbet')
    async def create_bet(self, ctx, description: str, option1: str, option2: str):
        """Créer un pari personnalisé."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = ctx.author.id
        bet_id = create_bet_db(user_id, description, option1, option2)
        await ctx.send(t("betting.created", lang, id=bet_id, desc=description))

    @commands.command(name='bet')
    async def place_bet(self, ctx, bet_id: str, choice: str, amount: int):
        """Parier sur un pari personnalisé."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        choice = choice.lower()
        bet_data = get_bet_data(bet_id)
        if not bet_data:
            return await ctx.send(t("betting.not_found", lang, id=bet_id))
        if bet_data["status"] == "CLOSE":
            return await ctx.send(t("betting.finished", lang))
        if bet_data["status"] == "FROZEN":
            return await ctx.send(t("betting.frozen", lang))
        current_balance = get_balance(ctx.author.id)
        if current_balance < amount:
            return await ctx.send(t("betting.no_money", lang, balance=current_balance))
        update_balance(ctx.author.id, -amount)
        choice_description = bet_data["options"][0] if choice == "a" else bet_data["options"][1]
        add_wager(bet_id, ctx.author.id, choice, amount)
        return await ctx.send(t("betting.placed", lang, choice=choice_description))

    @commands.command(name='closebet')
    async def close_bet(self, ctx, bet_id: str, winning_option: str):
        """Terminer un pari et distribuer les gains."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        bet_data = get_bet_data(bet_id)
        winning_option = winning_option.lower()
        if not bet_data:
            return await ctx.send(t("betting.id_not_found", lang))
        bet_id = bet_data["id"]
        if ctx.author.id != bet_data["creator"]:
            return await ctx.send(t("betting.only_creator", lang))
        if bet_data["status"] == "CLOSE":
            return await ctx.send(t("betting.already_closed", lang))
        if winning_option not in ["a", "b"]:
            return await ctx.send(t("betting.invalid_choice", lang))
            
        bet_winning_option = bet_data['options'][0] if winning_option == "a" else bet_data['options'][1]
        total_pool = sum(w["amount"] for w in bet_data["wagers"])
        winning_pool = sum(w["amount"] for w in bet_data["wagers"] if w["option"] == winning_option)
        results = []
        if winning_pool == 0:
            await ctx.send(t("betting.closed_house_keeps", lang, option=bet_winning_option, pool=total_pool))
        else:
            multiplier = total_pool / winning_pool
            for wager in bet_data["wagers"]:
                user_id = wager["user_id"]
                if wager["option"] == winning_option:
                    increment_stat(user_id, "wagers_won")
                    wager_amount = wager["amount"]
                    payout = int(wager_amount * multiplier)
                    update_balance(user_id, payout)
                    user = self.bot.get_user(user_id)
                    name = user.display_name if user else "Unknown"
                    results.append(t("betting.won_msg", lang, user=name, amount=payout))
                else:
                    increment_stat(user_id, "wagers_lost")
                
                unlocks = check_and_unlock_achievements(user_id)
                if unlocks:
                    user_obj = self.bot.get_user(user_id)
                    if user_obj:
                        await ctx.send(content=user_obj.mention, embed=format_achievements_unlocks(unlocks, lang))
            
            embed = discord.Embed(title=t("betting.result_title", lang, id=bet_id), description=f"{t('item_manager.winner', lang)}: **{bet_winning_option}**",
                                  color=discord.Color.purple())
            embed.add_field(name=t("betting.total_pool", lang), value=f"${total_pool}")
            embed.add_field(name=t("betting.winners", lang), value="\n".join(results) if results else "None")
            await ctx.send(embed=embed)
        close_bet_db(bet_id, winning_option)
        return None

    @commands.command(name='odds', aliases=['betinfo', 'status'])
    async def show_odds(self, ctx, bet_id: str):
        """Voir les cotes et infos d'un pari."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        bet_data = get_bet_data(bet_id)
        if not bet_data:
            return await ctx.send(t("inventory.item_not_found", lang).replace(t("inventory.item", lang), "bet"))
        
        bet_id = bet_data["id"]
        total_pool = sum(w["amount"] for w in bet_data["wagers"])
        opt1 = bet_data["options"][0]
        opt2 = bet_data["options"][1]
        pool_1 = sum(w["amount"] for w in bet_data["wagers"] if w["option"] == "a")
        pool_2 = sum(w["amount"] for w in bet_data["wagers"] if w["option"] == "b")
        odds_1 = round(total_pool / pool_1, 2) if pool_1 > 0 else "N/A"
        odds_2 = round(total_pool / pool_2, 2) if pool_2 > 0 else "N/A"

        embed = discord.Embed(title=t("betting.status_title", lang, id=bet_id), description=bet_data["description"],
                               color=discord.Color.blue())
        embed.add_field(name=t("betting.total_bet", lang), value=f"${total_pool}", inline=False)
        embed.add_field(
            name=t("betting.option_a", lang, name=opt1.capitalize()),
            value=f"**Value:** ${pool_1}\n**Odds:** {odds_1}x",
            inline=True
        )
        embed.add_field(
            name=t("betting.option_b", lang, name=opt2.capitalize()),
            value=f"**Value:** ${pool_2}\n**Odds:** {odds_2}x",
            inline=True
        )
        if bet_data["status"] == "CLOSED" or bet_data["status"] == "CLOSE":
            embed.set_footer(text=t("betting.odds_footer_closed", lang, winner=bet_data.get('winner', 'Unknown')))
        else:
            embed.set_footer(text=t("betting.odds_footer_open", lang))
        return await ctx.send(embed=embed)

    @commands.command(name="freezebet")
    async def freeze_bet(self, ctx, bet_id: str):
        """Geler un pari pour empêcher de nouvelles mises."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        bet_data = get_bet_data(bet_id)
        if not bet_data:
            return await ctx.send(t("betting.id_not_found", lang))
        if ctx.author.id != bet_data["creator"]:
            return await ctx.send(t("betting.only_creator", lang))
        if bet_data["status"] == "CLOSE":
            return await ctx.send(t("betting.already_closed", lang))
        if bet_data["status"] == "FROZEN":
            return await ctx.send(t("betting.frozen", lang))
        freeze_bet(bet_id)
        return await ctx.send(t("betting.freeze_success", lang, desc=bet_data["description"]))


async def setup(bot):
    await bot.add_cog(Betting(bot))