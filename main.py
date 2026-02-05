import asyncio
import logging
import os
import sys
import traceback
from venv import logger

import discord
from discord.ext import commands
from dotenv import load_dotenv

from src.globals import ITEMS_REGISTRY
from src.items.Beer import Beer
from src.items.CheatCoin import CheatCoin
from src.items.Coffee import Coffee
from src.items.FarmItem import Wheat, Oat, Potato, Tomato, Pumpkin, CocoaBean, Strawberry, GoldenApple, StarFruit, Corn, \
    RottenPlant
from src.items.FishingLoot import OldBoot, Trout, Salmon, Pufferfish, Swordfish, Sardine, KrakenTentacle, Carp, Whale, \
    Shark
from src.items.FortuneCookie import FortuneCookie
from src.items.LandDeed import VegetablePatchDeed, OrchardDeed, GreenhouseDeed
from src.items.Magnet import RustyMagnet, Magnet, ElectricMagnet
from src.items.MiningLoot import Emerald, PlatinumOre, GoldNugget, SilverOre, CopperOre, IronOre, Coal, Pebble, \
    Diamond
from src.items.ScratchTicket import ScratchTicket
from src.items.VipTicket import VipTicket

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


@bot.event
async def on_command_error(ctx, error):
    if isinstance(error, commands.CommandNotFound):
        return
    if isinstance(error, commands.CommandOnCooldown):
        await ctx.send(f"⏳ Doucement ! Réessaie dans {error.retry_after:.1f}s.")
        return
    if isinstance(error, commands.MissingRequiredArgument):
        await ctx.send(f"❌ Il manque des informations ! Vérifie la commande.")
        return
    print(f"\n⚠️ ERREUR DANS LA COMMANDE : {ctx.command}", file=sys.stderr)
    if isinstance(error, commands.CommandInvokeError):
        error = error.original
    traceback.print_exception(type(error), error, error.__traceback__, file=sys.stderr)


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
        name="⚒️ Les 3 Grands Métiers (XP)",
        value=(
            "`!mine` : Expédition minière. Gère ton risque pour trouver des diamants.\n"
            "`!fish` : Jeu de réflexe. Choisis ton biome et clique au bon moment !\n"
            "`!farm` : Gestion agricole. Achète des terrains et gère tes récoltes. \n"
            "`!level` : Vois les différents niveaux que t'as atteint \n"
        ),
        inline=False
    )

    embed.add_field(
        name="🎒 Inventaire & Craft",
        value=(
            "`!inv` (ou `!bag`) : Voir tes ressources et objets.\n"
            "`!craft` : Oouvre l'atelier pour fabriquer des objets (Pain, Outils...).\n"
            "`!use <objet>` : Utiliser un consommable (Potion, Dynamite...).\n"
            "`!sell <@joueur> <objet> <valeur>` : Vend un objet à un autre joueur."

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
        name="📈 Marché",
        value=(
            "`!market` : Voir le cours de la bourse (Krach ou Boom ?).\n"
            "`!market_sell <item> [qte]` : Vendre tes ressources au prix du marché.\n"
            # "`!shop` : Acheter des permis, des terrains ou des objets.\n"
            "`!daily` : Ton salaire de base."
        ),
        inline=False
    )

    embed.add_field(
        name="🎒 Inventaire & Boutique",
        value=(
            # "`!shop` : Acheter des objets et licences.\n"
            "`!inventory` (ou `!inv`) : Voir ton sac.\n"
            "`!use <item>` : Utiliser un objet.\n"
            # "`!licenses` : Voir les spécialisations (Mineur Pro, VIP...)."
        ),
        inline=False
    )

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
    embed.set_footer(text=f"Astuce : Monte de niveau dans ton métier pour gagner plus !")
    return await ctx.send(embed=embed)


@bot.event
async def on_ready():
    print(f'Logged in as {bot.user.name}')
    print('System is online.')

def initialize_items():

    # Consumable
    Coffee().register()
    VipTicket().register()
    Beer().register()
    FortuneCookie().register()
    CheatCoin().register()
    Magnet().register()
    RustyMagnet().register()
    ElectricMagnet().register()
    ScratchTicket().register()


    # Mining resources
    Pebble().register()
    Coal().register()
    IronOre().register()
    CopperOre().register()
    SilverOre().register()
    GoldNugget().register()
    PlatinumOre().register()
    Emerald().register()
    Diamond().register()

    # Fishing resources
    OldBoot().register()
    Trout().register()
    Salmon().register()
    Pufferfish().register()
    Swordfish().register()
    Sardine().register()
    KrakenTentacle().register()
    Carp().register()
    Shark().register()
    Whale().register()

    # Farm Items
    Wheat().register()
    Oat().register()
    Corn().register()
    Potato().register()
    Tomato().register()
    Pumpkin().register()
    CocoaBean().register()
    Strawberry().register()
    GoldenApple().register()
    StarFruit().register()
    RottenPlant().register()

    # Lands to buy
    VegetablePatchDeed().register()
    GreenhouseDeed().register()
    OrchardDeed().register()


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
