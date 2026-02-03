import asyncio
import logging
import os
import sys
from venv import logger

import discord
from discord.ext import commands
from dotenv import load_dotenv

from src.globals import ITEMS_REGISTRY
from src.items.Coffee import Coffee
from src.items.Ticket import Ticket

load_dotenv()
TOKEN = os.getenv('DISCORD_TOKEN')


intents = discord.Intents.default()
intents.message_content = True
intents.members = True
bot = commands.Bot(command_prefix='!', intents=intents)
handler = logging.FileHandler(filename='discord.log', encoding='utf-8', mode='w')
logger.addHandler(handler)

@bot.event
async def on_ready():
    print(f'Logged in as {bot.user.name}')
    print('GuacaGambleBot is online.')


async def load_extensions():
    for ext in os.listdir("src/cogs"):
        if ext.endswith(".py") and not ext.startswith("_"):
            try:
                await bot.load_extension(f"src.cogs.{ext[:-3]}")
                print(f"Loaded {ext}")
            except Exception as e:
                print(f"Failed to load {ext}: {e}")


bot.remove_command('help')



@bot.command(name='help')
async def help_command(ctx):
    """Affiche le guide des commandes en français."""
    embed = discord.Embed(
        title="📚 Aide - GuacaGambleBot",
        description="Pariez de l'argent virtuel et devenez le plus riche du serveur !",
        color=discord.Color.teal()
    )
    # --- 1. ÉCONOMIE ---
    embed.add_field(
        name="💰 Économie",
        value=(
            "`!daily` : Ton salaire journalier.\n"
            "`!balance` (ou `!bal`) : Voir ton solde.\n"
            "`!give <@joueur> <montant>` : Faire un virement."
        ),
        inline=False
    )

    embed.add_field(
        name="🎰 Casino & Cartes",
        value=(
            "`!slots <mise>` : Machine à sous. Vise les 💎 !\n"

            "`!coinflip <pile/face> <mise>` : Quitte ou double rapide.\n"
            "`!lotto` : Voir la cagnotte du loto."
        ),
        inline=False
    )

    embed.add_field(
        name="🔫 Duels & Adrénaline",
        value=(
            "`!duel <@joueur> <mise>` : Provoque quelqu'un en duel (50/50).\n"
            "`!blackjack <mise>` (ou `!bjduel`) : Le 21. Affronte un autre joueur.\n"
            "`!roulette <mise>` : Roulette Russe. 1 chance sur 6 de mourir (et perdre la mise).\n"
            "*Note : Le duel nécessite que l'adversaire accepte.*"
        ),
        inline=False
    )

    embed.add_field(
        name="⛏️ Métiers (Farming) **NON IMPLÉMENTÉ**",
        value=(
            "`!mine` : Mine des ressources pour l'XP et l'argent.\n"
            "`!fish` : Pêche des poissons (consommables).\n"
            "`!jobstats` : Voir ta progression de niveau."
        ),
        inline=False
    )

    embed.add_field(
        name="🎲 Paris",
        value=(
            "`!bet <ID> <Choix> <Montant>`: Place un pari. *Ex: !bet 1 A 100*\n"
            "`!odds <ID>`: Affiche les cotes et la cagnotte d'un pari.\n"
            "`!createbet \"Question\" \"A\" \"B\"`: Créer un pari. *N'oubliez pas les guillemets !*\n"
            "`!closebet <ID> <Gagnant>`: Terminer un pari et payer les vainqueurs.\n"
            "`!freezebet <ID>`: Gêle la possibilité de parier sur un pari."
        ),
        inline=False)

    embed.add_field(
        name="🎒 Inventaire & Boutique",
        value=(
            # "`!shop` : Acheter des objets et licences.\n"
            # "`!inventory` (ou `!inv`) : Voir ton sac.\n"
            "`!use <item>` : Utiliser un objet.\n"
            # "`!licenses` : Voir les spécialisations (Mineur Pro, VIP...)."
        ),
        inline=False
    )

    # --- 6. BANQUE ---
    embed.add_field(
        name="🏦 Banque & Prêts",
        value=(
            "!deposit <montant>: (ou !dep) Dépose de l'argent dans ta banque (max 500)\n"
            "!withdraw <montant>: (ou !wd) Retire l'argent de ta banque vers ton portefeuille\n"
            "`!lend <@joueur> <montant>` : Prêter avec 10% d'intérêt.\n"
            "`!repay <montant>` : Rembourser tes dettes. (réparti entre tes différents créantiers)\n"
            "`!debt` : Voir ce que tu dois aux autres."
        ),
        inline=False
    )
    embed.set_footer(text="Bonne chance à tous ! 🎰")
    return await ctx.send(embed=embed)


@bot.event
async def on_ready():
    print(f'Logged in as {bot.user.name}')
    print('System is online.')

def initialize_items():
    """Crée les instances et les enregistre dans la DB."""
    Coffee().register()
    Ticket().register()
    print(f"✅ {len(ITEMS_REGISTRY)} objets chargés dans le système.")

async def main():
    initialize_items()
    async with bot:
        await load_extensions()
        await bot.start(TOKEN)


if __name__ == '__main__':
    if not TOKEN:
        print("Error: DISCORD_TOKEN not found in .env")
    else:
        asyncio.run(main())
