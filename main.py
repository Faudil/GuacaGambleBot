import asyncio
import logging
import os
import sys
import traceback
from venv import logger

import discord
from discord.ext import commands
from dotenv import load_dotenv

from src.globals import ITEMS_REGISTRY, TEST_CHANNEL_ID
from src.items.Beer import Beer
from src.items.Bow import Bow
from src.items.CheatCoin import CheatCoin
from src.items.Coffee import Coffee
from src.items.FarmItem import Wheat, Oat, Potato, Tomato, Pumpkin, CocoaBean, Strawberry, GoldenApple, StarFruit, Corn, \
    RottenPlant
from src.items.Fertilizer import Fertilizer
from src.items.FishingLoot import OldBoot, Trout, Salmon, Pufferfish, Swordfish, Sardine, KrakenTentacle, Carp, Whale, \
    Shark
from src.items.FortuneCookie import FortuneCookie
from src.items.Hook import Hook
from src.items.LandDeed import VegetablePatchDeed, OrchardDeed, GreenhouseDeed
from src.items.Magnet import RustyMagnet, Magnet, ElectricMagnet
from src.items.MiningLoot import Emerald, PlatinumOre, GoldNugget, SilverOre, CopperOre, IronOre, Coal, Pebble, \
    Diamond
from src.items.MysteryEgg import MysteryEgg
from src.items.ScratchTicket import ScratchTicket
from src.items.VipTicket import VipTicket
from src.globals import CHANNEL_ID

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
    """Affiche le guide des commandes."""
    embed = discord.Embed(
        title="📚 Aide - GuacaGambleBot",
        description="Pariez de l'argent virtuel et devenez le plus riche du serveur !\nVoici la liste de toutes les commandes disponibles :",
        color=discord.Color.teal()
    )

    for cog_name, cog in bot.cogs.items():
        if cog_name.startswith('_') or cog_name == 'Tasks':
            continue
            
        cog_commands = cog.get_commands()
        if not cog_commands:
            continue
            
        command_list = []
        for cmd in cog_commands:
            if not cmd.hidden:
                desc = cmd.short_doc or "Aucune description"
                command_list.append(f"`!{cmd.name}` : {desc}")
                
        if command_list:
            embed.add_field(
                name=f"📌 {cog_name}",
                value="\n".join(command_list),
                inline=False
            )

    uncogged_commands = [cmd for cmd in bot.commands if cmd.cog is None and cmd.name != "help" and not cmd.hidden]
    if uncogged_commands:
        command_list = []
        for cmd in uncogged_commands:
            desc = cmd.short_doc or "Aucune description"
            command_list.append(f"`!{cmd.name}` : {desc}")
        
        if command_list:
            embed.add_field(
                name="⚙️ Autres commandes",
                value="\n".join(command_list),
                inline=False
            )

    embed.set_footer(text="Astuce : Utilisez !help <commande> pour plus de détails (si implémenté) !")
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
    Hook().register()
    Fertilizer().register()
    FortuneCookie().register()
    CheatCoin().register()
    Magnet().register()
    RustyMagnet().register()
    ElectricMagnet().register()
    ScratchTicket().register()
    Bow().register()


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

    # Pet
    MysteryEgg().register()

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
