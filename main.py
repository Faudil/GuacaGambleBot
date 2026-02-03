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
    embed.add_field(
        name="💰 Économie",
        value=(
            "**`!daily`**\nRécupère ton salaire quotidien (50$).\n"

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
            "**`!createbet \"Question\" \"A\" \"B\"`**\nCréer un pari. *N'oubliez pas les guillemets !*\n"
            "**`!closebet <ID> <Gagnant>`**\nTerminer un pari et payer les vainqueurs.\n"
            "**`!freezebet <ID>`**\nGêle la possibilité de parier sur un pari."
        ),
        inline=False
    )
    embed.add_field(
        name="🎰 Gambling et Casino",
        value=(
            "**`!coinflip`** (ou `!pileouface`)\n Pile ou face *Ex: !coinflip pile 50.*\n"
            "**`!casino`** (ou `!slots`)\n Tente ta chance et augmente jusqu'à 10 fois ta mise *Ex: !casino 10*.\n"
            "**`!blackjack`** (ou `!bjduel`)\n Defi un autre joueur sur le serveur *Ex: !casino 10*.\n"
            "**`!duel`**\n Defi un autre joueur sur le serveur *Ex: !duel @joueur *.\n"
    ), inline=False)
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
