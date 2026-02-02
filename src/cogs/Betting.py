import discord
from discord.ext import commands
from src.data_handling import update_balance, get_balance, create_bet_db, add_wager, get_bet_data, close_bet_db, \
    freeze_bet


class Betting(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name='createbet')
    async def create_bet(self, ctx, description: str, option1: str, option2: str):
        user_id = ctx.author.id
        bet_id = create_bet_db(user_id, description, option1, option2)
        await ctx.send(f"🎲 Pari #{bet_id} créé: {description}")

    @commands.command(name='bet')
    async def place_bet(self, ctx, bet_id: str, choice: str, amount: int):
        choice = choice.lower()
        bet_data = get_bet_data(bet_id)
        if not bet_data:
            return await ctx.send(f"❌ Paris {bet_id}.")
        if bet_data["status"] == "CLOSE":
            return await ctx.send("❌ Le pari est terminé.")
        if bet_data["status"] == "FROZEN":
            return await ctx.send("❌ Le pari est gelé.")
        current_balance = get_balance(ctx.author.id)
        if current_balance < amount:
            return await ctx.send(f"❌ T'as pas assez d'argent (tu as {current_balance}).")
        update_balance(ctx.author.id, -amount)
        choice_description = bet_data["options"][0] if choice == "a" else bet_data["options"][1]
        add_wager(bet_id, ctx.author.id, choice, amount)
        return await ctx.send(f"✅ Bet placed on {choice_description}!")

    @commands.command(name='closebet')
    async def close_bet(self, ctx, bet_id: str, winning_option: str):
        """End a bet and payout. Usage: !closebet <id> <winning_option>"""
        bet_data = get_bet_data(bet_id)
        winning_option = winning_option.lower()
        if not bet_data:
            return await ctx.send("❌ ID du pari non trouvé.")
        bet_id = bet_data["id"]
        if ctx.author.id != bet_data["creator"]:
            return await ctx.send("❌ Seul le créateur peut fermer le pari.")
        if bet_data["status"] == "CLOSE":
            return await ctx.send("❌ Ce pari a déjà été fermé.")
        if winning_option not in ["a", "b"]:
            return await ctx.send(f"❌ Choix invalide. Choisis: A ou B")
        bet_winning_option = bet_data['options'][0] if winning_option == "a" else bet_data['options'][1]
        total_pool = sum(w["amount"] for w in bet_data["wagers"])
        winning_pool = sum(w["amount"] for w in bet_data["wagers"] if w["option"] == winning_option)
        results = []
        if winning_pool == 0:
            await ctx.send(
                f"🔒 pari fermé. **{bet_winning_option}** a gagné, mais personne n'a parié dessus. La maison garde ${total_pool}!")
        else:
            multiplier = total_pool / winning_pool
            for wager in bet_data["wagers"]:
                if wager["option"] == winning_option:
                    user_id = wager["user_id"]
                    wager_amount = wager["amount"]
                    payout = int(wager_amount * multiplier)
                    update_balance(user_id, payout)
                    user = self.bot.get_user(user_id)
                    name = user.display_name if user else "Unknown"
                    results.append(f"{name} won ${payout}")
            embed = discord.Embed(title=f"🏆 pari #{bet_id} Résultat", description=f"Gagnant: **{bet_winning_option}**",
                                  color=discord.Color.purple())
            embed.add_field(name="Valeur total", value=f"${total_pool}")
            embed.add_field(name="Gagnants", value="\n".join(results) if results else "None")
            await ctx.send(embed=embed)
        close_bet_db(bet_id, winning_option)
        return None

    @commands.command(name='odds', aliases=['betinfo', 'status'])
    async def show_odds(self, ctx, bet_id: str):
        """Check the current pool and odds for a specific bet."""
        bet_data = get_bet_data(bet_id)
        if not bet_data:
            return await ctx.send("❌ Je connais pas ce pari chef !")
        bet_id = bet_data["id"]
        total_pool = sum(w["amount"] for w in bet_data["wagers"])
        opt1 = bet_data["options"][0]
        opt2 = bet_data["options"][1]
        pool_1 = sum(w["amount"] for w in bet_data["wagers"] if w["option"] == "a")
        pool_2 = sum(w["amount"] for w in bet_data["wagers"] if w["option"] == "b")
        odds_1 = round(total_pool / pool_1, 2) if pool_1 > 0 else "N/A"
        odds_2 = round(total_pool / pool_2, 2) if pool_2 > 0 else "N/A"

        embed = discord.Embed(title=f"📊 Statut: pari #{bet_id}", description=bet_data["description"],
                              color=discord.Color.blue())
        embed.add_field(name="💰 Total parié", value=f"${total_pool}", inline=False)
        embed.add_field(
            name=f"Option A: {opt1.capitalize()}",
            value=f"**Valeur:** ${pool_1}\n**Cote:** {odds_1}x",
            inline=True
        )
        embed.add_field(
            name=f"Option B: {opt2.capitalize()}",
            value=f"**Valeur:** ${pool_2}\n**Cote:** {odds_2}x",
            inline=True
        )
        if bet_data["status"] == "CLOSED":
            embed.set_footer(text=f"Ce pari est terminé. Gagnant: {bet_data.get('winner', 'Unknown')}")
        else:
            embed.set_footer(text=f"Statut: OUVERT | La cote change au fur et à mesure que les gens parient!")
        return await ctx.send(embed=embed)

    @commands.command(name="freezebet")
    async def freeze_bet(self, ctx, bet_id: str):
        """Freeze the bet and stop . Usage: !freezebet <bet_id>"""
        bet_data = get_bet_data(bet_id)
        if bet_id not in bet_data["bets"]:
            return await ctx.send("❌ ID du pari non trouvé.")
        if ctx.author.id != bet_data["creator"]:
            return await ctx.send("❌ Seul le créateur peut geler le pari.")
        if bet_data["status"] == "CLOSE":
            return await ctx.send("❌ Ce pari a déjà été fermé.")
        if bet_data["status"] == "FROZEN":
            return await ctx.send("❌ Ce pari a déjà été fermé.")
        freeze_bet(bet_id)
        return await ctx.send(f'✅ Le pari {bet_data["description"]} a été gelé, vous ne pouvez plus parier dessus')


async def setup(bot):
    await bot.add_cog(Betting(bot))