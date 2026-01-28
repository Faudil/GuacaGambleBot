from datetime import datetime, timedelta
import json
import os
import random

import discord
from discord.ext import commands
from dotenv import load_dotenv

load_dotenv()
TOKEN = os.getenv('DISCORD_TOKEN')
DATA_FILE = 'data/betting_data.json' # MongoDB at home lol
STARTING_BALANCE = 100
DAILY_AMOUNT = 50

intents = discord.Intents.default()
intents.message_content = True
intents.members = True
bot = commands.Bot(command_prefix='!', intents=intents)


def load_data():
    if not os.path.exists(DATA_FILE):
        return {"users": {}, "bets": {}}
    with open(DATA_FILE, 'r') as f:
        return json.load(f)


def save_data(data):
    with open(DATA_FILE, 'w') as f:
        json.dump(data, f, indent=4)


def get_balance(user_id):
    data = load_data()
    user_id = str(user_id)
    if user_id not in data["users"]:
        data["users"][user_id] = STARTING_BALANCE
        save_data(data)
    return data["users"][user_id]


def update_balance(user_id, amount):
    data = load_data()
    user_id = str(user_id)
    if user_id not in data["users"]:
        data["users"][user_id] = STARTING_BALANCE
    data["users"][user_id] += amount
    save_data(data)
    return data["users"][user_id]


@bot.event
async def on_ready():
    print(f'Logged in as {bot.user.name}')
    print('GuacaGambleBot is online.')



@bot.command(name='balance', aliases=['bal', 'money', 'solde'])
async def balance(ctx, member: discord.Member = None):
    """Check your current balance."""
    if member is None:
        target = ctx.author
    else:
        target = member
    bal = get_balance(target.id)
    embed = discord.Embed(title="💰 Compte en banque", color=discord.Color.green())
    embed.add_field(name="User", value=target.display_name)
    embed.add_field(name="Balance", value=f"${bal}")
    await ctx.send(embed=embed)


@bot.command(name='daily')
async def daily(ctx):
    """Collect your daily allowance (Once every 24 hours)."""
    data = load_data()
    user_id = str(ctx.author.id)
    now = datetime.now()

    if "last_daily" not in data:
        data["last_daily"] = {}

    if user_id in data["last_daily"]:
        last_claim_str = data["last_daily"][user_id]
        last_claim = datetime.fromisoformat(last_claim_str)

        time_diff = now - last_claim

        if time_diff < timedelta(hours=24):
            time_left = timedelta(hours=24) - time_diff
            hours, remainder = divmod(int(time_left.total_seconds()), 3600)
            minutes, _ = divmod(remainder, 60)

            return await ctx.send(
                f"⏳ **{ctx.author.display_name}**, t'as déjà demandé bouffon "
                f"**{hours}h {minutes}m** Pour ta prochaine paie."
            )

    if user_id not in data["users"]:
        data["users"][user_id] = STARTING_BALANCE

    data["users"][user_id] += DAILY_AMOUNT
    current_bal = data["users"][user_id]

    data["last_daily"][user_id] = now.isoformat()
    save_data(data)

    embed = discord.Embed(title="💸 Voilà ta thune", color=discord.Color.green())
    embed.add_field(name="Quantité", value=f"+${DAILY_AMOUNT}")
    embed.add_field(name="Ta balance", value=f"${current_bal}")
    embed.set_footer(text="Reviens dans 24h !")

    return await ctx.send(embed=embed)


@bot.command(name='createbet')
async def create_bet(ctx, description: str, option1: str, option2: str):
    """Create a new bet. Usage: !createbet "Will it rain?" "Yes" "No" """
    data = load_data()
    bet_id = str(len(data["bets"]) + 1)

    bet_structure = {
        "id": bet_id,
        "creator": ctx.author.id,
        "description": description,
        "options": [option1, option2],
        "frozen": False,
        "status": "OPEN",
        "wagers": []
    }

    data["bets"][bet_id] = bet_structure
    save_data(data)
    embed = discord.Embed(title=f"🎲 Pari #{bet_id} créé !", description=description, color=discord.Color.gold())
    embed.add_field(name="Option A", value=option1, inline=True)
    embed.add_field(name="Option B", value=option2, inline=True)
    embed.set_footer(text=f"Utilise: !bet {bet_id} <option> <amount>")

    await ctx.send(embed=embed)


@bot.command(name="freezebet")
async def freeze_bet(ctx, bet_id: str):
    """Freeze the bet and stop . Usage: !freezebet <bet_id>"""
    data = load_data()
    if bet_id not in data["bets"]:
        return await ctx.send("❌ ID du pari non trouvé.")
    bet = data["bets"][bet_id]
    if ctx.author.id != bet["creator"]:
        return await ctx.send("❌ Seul le créateur peut geler le pari.")
    if bet["status"] != "OPEN":
        return await ctx.send("❌ Ce pari a déjà été fermé.")
    if "frozen" in bet and bet["frozen"]:
        return await ctx.send("❌ Ce pari a déjà été gelé.")
    bet["frozen"] = True
    save_data(data)
    return await ctx.send(f'✅ Le pari {bet["description"]} a été gelé, vous ne pouvez plus parier dessus')



@bot.command(name='bet')
async def place_bet(ctx, bet_id: str, choice: str, amount: int):
    """Place a bet. Usage: !bet <id> <choice> <amount>"""
    data = load_data()
    choice = choice.lower()
    if bet_id not in data["bets"]:
        return await ctx.send("❌ Je trouve pas le pari chef !")
    bet = data["bets"][bet_id]
    if bet["status"] != "OPEN":
        return await ctx.send("❌ Ce pari est terminé")
    if bet["frozen"]:
        return await ctx.send("❌ Ce pari a été gelé, vous ne pouvez plus parier dessus")
    if choice not in ["a", "b"]:
        return await ctx.send(f"❌ J'ai pas compris, essaie plutôt: A or B")
    if amount <= 0:
        return await ctx.send("❌ Tu dois au moins parier $1.")
    user_bal = get_balance(ctx.author.id)
    if user_bal < amount:
        return await ctx.send(f"❌ T'as pas assez de tal. Tu as actuellement ${user_bal}.")
    update_balance(ctx.author.id, -amount)
    bet_choice = bet['options'][0] if choice == "a" else bet['options'][1]
    wager = {
        "user_id": ctx.author.id,
        "option": bet_choice,
        "amount": amount
    }
    data = load_data()
    data["bets"][bet_id]["wagers"].append(wager)
    save_data(data)
    return await ctx.send(f"✅ **{ctx.author.display_name}** a parié **${amount}** sur **{bet_choice}** Pour le pari #{bet_id}!")



@bot.command(name='odds', aliases=['betinfo', 'status'])
async def show_odds(ctx, bet_id: str):
    """Check the current pool and odds for a specific bet."""
    data = load_data()
    if bet_id not in data["bets"]:
        return await ctx.send("❌ Je connais pas ce pari chef !")
    bet = data["bets"][bet_id]
    total_pool = sum(w["amount"] for w in bet["wagers"])
    opt1 = bet["options"][0]
    opt2 = bet["options"][1]
    pool_1 = sum(w["amount"] for w in bet["wagers"] if w["option"] == opt1)
    pool_2 = sum(w["amount"] for w in bet["wagers"] if w["option"] == opt2)
    odds_1 = round(total_pool / pool_1, 2) if pool_1 > 0 else "N/A"
    odds_2 = round(total_pool / pool_2, 2) if pool_2 > 0 else "N/A"

    embed = discord.Embed(title=f"📊 Statut: pari #{bet_id}", description=bet["description"], color=discord.Color.blue())
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
    if bet["status"] == "CLOSED":
        embed.set_footer(text=f"Ce pari est terminé. Gagnant: {bet.get('winner', 'Unknown')}")
    else:
        embed.set_footer(text=f"Statut: OUVERT | La cote change au fur et à mesure que les gens parient!")
    return await ctx.send(embed=embed)



@bot.command(name='closebet')
async def close_bet(ctx, bet_id: str, winning_option: str):
    """End a bet and payout. Usage: !closebet <id> <winning_option>"""
    data = load_data()
    winning_option = winning_option.lower()
    if bet_id not in data["bets"]:
        return await ctx.send("❌ ID du pari non trouvé.")
    bet = data["bets"][bet_id]
    if ctx.author.id != bet["creator"]:
        return await ctx.send("❌ Seul le créateur peut fermer le pari.")
    if bet["status"] != "OPEN":
        return await ctx.send("❌ Ce pari a déjà été fermé.")
    if winning_option not in ["a", "b"]:
        return await ctx.send(f"❌ Choix invalide. Choisis: A ou B")
    bet_winning_option = bet['options'][0] if winning_option == "a" else bet['options'][1]
    total_pool = sum(w["amount"] for w in bet["wagers"])
    winning_pool = sum(w["amount"] for w in bet["wagers"] if w["option"] == bet_winning_option)
    results = []
    if winning_pool == 0:
        await ctx.send(f"🔒 pari fermé. **{bet_winning_option}** a gagné, mais personne n'a parié dessus. La maison garde ${total_pool}!")
    else:
        multiplier = total_pool / winning_pool
        for wager in bet["wagers"]:
            if wager["option"] == bet_winning_option:
                user_id = wager["user_id"]
                wager_amount = wager["amount"]
                payout = int(wager_amount * multiplier)
                update_balance(user_id, payout)
                user = bot.get_user(user_id)
                name = user.display_name if user else "Unknown"
                results.append(f"{name} won ${payout}")
        embed = discord.Embed(title=f"🏆 pari #{bet_id} Résultat", description=f"Gagnant: **{bet_winning_option}**",
                              color=discord.Color.purple())
        embed.add_field(name="Valeur total", value=f"${total_pool}")
        embed.add_field(name="Gagnants", value="\n".join(results) if results else "None")
        await ctx.send(embed=embed)
    data = load_data()
    data["bets"][bet_id]["status"] = "CLOSED"
    data["bets"][bet_id]["winner"] = bet_winning_option
    save_data(data)
    return None


@bot.command(name='top', aliases=['leaderboard', 'classement'])
async def leaderboard(ctx):
    """Display the 5 richest in the server."""
    data = load_data()
    users = data.get("users", {})
    if not users:
        return await ctx.send("Personne n'a d'argent pour l'instant !")
    sorted_users = sorted(users.items(), key=lambda x: x[1], reverse=True)
    top_5 = sorted_users[:5]
    embed = discord.Embed(title="🏆 Classement des plus riches", color=discord.Color.gold())
    description = ""
    for i, (user_id, balance) in enumerate(top_5, 1):
        member = ctx.guild.get_member(int(user_id))
        name = member.display_name if member else "Inconnu"
        medals = {1: "🥇", 2: "🥈", 3: "🥉"}
        rank_emoji = medals.get(i, "🔹")
        description += f"{rank_emoji} **{name}** : ${balance}\n"
    embed.description = description
    return await ctx.send(embed=embed)


@bot.command(name='coinflip', aliases=['cf', 'pileouface'])
async def coinflip(ctx, choice: str, amount: int):
    """Bet against the bot. Usage: !cf <pile/face> <amount>"""
    data = load_data()
    user_id = str(ctx.author.id)
    choice = choice.lower()
    valid_choices = ["pile", "face"]
    today_str = datetime.now().strftime("%Y-%m-%d")
    if "cf_limits" not in data:
        data["cf_limits"] = {}
    user_limit = data["cf_limits"].get(user_id, {"date": "", "count": 0})
    if user_limit["date"] != today_str:
        user_limit = {"date": today_str, "count": 0}
    if user_limit["count"] >= 10:
        return await ctx.send(
            f"🛑 **{ctx.author.display_name}**, tu as atteint ta limite de 10 paris pour aujourd'hui ! Reviens demain.")
    if choice not in valid_choices:
        return await ctx.send("❌ Choisis **pile** ou **face**.")
    if amount <= 0:
        return await ctx.send("❌ Tu dois parier plus que 0$.")
    user_bal = data["users"].get(user_id, STARTING_BALANCE)
    if user_bal < amount:
        return await ctx.send(f"❌ Tu n'as pas assez d'argent (${user_bal}).")
    user_limit["count"] += 1
    data["cf_limits"][user_id] = user_limit
    outcome = random.choice(valid_choices)
    remaining = 10 - user_limit["count"]
    save_data(data)
    if choice == outcome:
        update_balance(user_id, amount)
        return await ctx.send(f"🪙 La pièce tombe sur **{outcome.upper()}** ! Tu gagnes **${amount}** ! Tu possèdes maintenant {user_bal + amount}$. Il te reste {remaining} coinflip pour aujourd'hui. 🎉")
    else:
        update_balance(user_id, -amount)
        return await ctx.send(f"🪙 La pièce tombe sur **{outcome.upper()}**... Tu perds **${amount}**. Tu possèdes maintenant {user_bal - amount}$. Il te reste {remaining} coinflip pour aujourd'hui. 😢")


@bot.command(name='give', aliases=['pay', 'transfer', 'donner'])
async def give(ctx, recipient: discord.Member, amount: int):
    """Transfer money to another user. Usage: !give @User 100"""
    sender_id = ctx.author.id
    recipient_id = recipient.id
    if sender_id == recipient_id:
        return await ctx.send("❌ Wesh frerot tu testes quoi là ?")
    if recipient.bot:
        return await ctx.send("❌ Tu peux pas donner de la thune à un bot.")
    if amount <= 0:
        return await ctx.send("❌ Tu dois donner plus de $0.")
    sender_bal = get_balance(sender_id)
    if sender_bal < amount:
        return await ctx.send(f"❌ You don't have enough money. Balance: ${sender_bal}")
    update_balance(sender_id, -amount)
    update_balance(recipient_id, amount)
    embed = discord.Embed(title="💸 Transaction complète", color=discord.Color.green())
    embed.add_field(name="Donneur", value=ctx.author.display_name, inline=True)
    embed.add_field(name="Receveur", value=recipient.display_name, inline=True)
    embed.add_field(name="Quantité", value=f"**${amount}**", inline=False)
    return await ctx.send(embed=embed)



bot.remove_command('help')

@bot.command(name='help')
async def help_command(ctx):
    """Affiche le guide des commandes en français."""
    embed = discord.Embed(
        title="📚 Aide - GuacaGambleBot",
        description="Pariez de l'argent virtuel et devenez le plus riche du serveur !",
        color=discord.Color.teal()
    )
    embed.add_field(
        name="💰 Économie",
        value=(
            "**`!daily`**\nRécupère ton salaire quotidien (50$).\n"
            "**`!coinflip`** (ou `!pileouface`)\n Pile ou face *Ex: !coinflip pile 50.*\n"
            "**`!balance`** (ou `!bal`)\nAffiche ton solde actuel.\n"
            "**`!top`** (ou `!classement`)\nAffiche le top 5 des plus riches.\n"
            "**`!give`** (ou `!donner`)\n Donne de l'argent à quelqu'un. *Ex: !give @guacamole 150*"
        ),
        inline=False
    )
    embed.add_field(
        name="🎲 Paris",
        value=(
            "**`!bet <ID> <Choix> <Montant>`**\nPlace un pari. *Ex: !bet 1 A 100*\n"
            "**`!odds <ID>`**\nAffiche les cotes et la cagnotte d'un pari.\n"

        ),
        inline=False
    )
    embed.add_field(
        name="👑 Organisation",
        value=(
            "**`!createbet \"Question\" \"A\" \"B\"`**\nCréer un pari. *N'oubliez pas les guillemets !*\n"
            "**`!closebet <ID> <Gagnant>`**\nTerminer un pari et payer les vainqueurs.\n"
            "**`!freezebet <ID>`**\nGêle la possibilité de parier sur un pari."
        ),
        inline=False
    )
    embed.set_footer(text="Bonne chance à tous ! 🎰")
    return await ctx.send(embed=embed)


if __name__ == '__main__':
    bot.run(TOKEN)
