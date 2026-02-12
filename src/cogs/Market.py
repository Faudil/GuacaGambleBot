import itertools

import discord
from discord.ext import commands, tasks
import random

from src.database.balance import update_balance
from src.database.item import remove_item_from_inventory, get_item_name_by_id, get_item_id_from_name, \
    get_all_user_inventory, has_item
from src.globals import ITEMS_REGISTRY
from src.items.FarmItem import Wheat, Oat, Corn, Tomato, Pumpkin, Potato, CoffeeBean, CocoaBean, Strawberry, \
    GoldenApple, StarFruit, RottenPlant
from src.items.MiningLoot import Coal, IronOre, GoldNugget, Diamond, Pebble, SilverOre, CopperOre, Emerald, \
    PlatinumOre
from src.items.FishingLoot import OldBoot, Trout, Salmon, Pufferfish, Swordfish, Sardine, KrakenTentacle, Carp, Whale, \
    Shark


class Market(commands.Cog):
    def __init__(self, bot):
        self.mining_items = [Pebble(), Coal(), IronOre(), CopperOre(),
            SilverOre(), GoldNugget(), PlatinumOre(),
            Emerald(), Diamond()]
        self.fishing_items = [
            OldBoot(), Trout(), Salmon(),
            Pufferfish(), Swordfish(), Sardine(),
            KrakenTentacle(), Carp(), Whale(),
            Shark(),

        ]
        self.farming_items = [
            Wheat(), Oat(), Corn(), Potato(), Tomato(),
            Pumpkin(), CoffeeBean(), CocoaBean(),
            Strawberry(), GoldenApple(), StarFruit(),
            RottenPlant()
        ]
        self.titles = [
            "📈⛏️ Cours items minages",
            "🦈 Cours items pêche",
            "📈🚜 Cours items ferme",
        ]
        self.total_items_nbr = len(self.mining_items) + len(self.fishing_items) + len(self.farming_items)
        self.sellable_items = [self.mining_items, self.fishing_items, self.farming_items]
        self.sellable_items_names = [item.name.lower() for item in itertools.chain(*self.sellable_items)]
        self.item_multipliers = [1] * self.total_items_nbr
        self.bot = bot
        self.market_multiplier = 1.0
        self.trend = "stable"
        self.update_market_prices.start()

    @tasks.loop(hours=4)
    async def market_event(self):
        random.random()


    @tasks.loop(minutes=5)
    async def update_market_prices(self):
        for i in range(0, len(self.item_multipliers)):
            change = random.choice([-0.1, -0.05, 0, 0.05, 0.1])
            multiplier = self.item_multipliers[i]
            multiplier += change
            multiplier = max(0.1, min(3, multiplier))
            self.item_multipliers[i] = multiplier
            self.market_multiplier = self.market_multiplier * multiplier

    @commands.command(name='market')
    async def show_market(self, ctx, to_show: str = None):
        mine_tag = ["mine", "minage", "mining"]
        fish_tag = ["fish", "pêche", "fishing"]
        farm_tag = ["farm", "ferme", "farming"]
        to_show_items = self.sellable_items
        to_show_items_titles = self.titles
        idx = 0
        if to_show in mine_tag:
            to_show_items = [self.mining_items]
            to_show_items_titles = [self.titles[0]]
        elif to_show in fish_tag:
            idx = len(self.mining_items)
            to_show_items = [self.fishing_items]
            to_show_items_titles = [self.titles[1]]
        elif to_show in farm_tag:
            idx = len(self.mining_items) + len(self.fishing_items)
            to_show_items = [self.farming_items]
            to_show_items_titles = [self.titles[2]]

        for titles, items in zip(to_show_items_titles, to_show_items):
            embed = discord.Embed(title=titles, color=discord.Color.gold())
            for item in items:
                item_id = get_item_id_from_name(item.name)
                multiplier = self.item_multipliers[idx]
                current_price = int(max(1, item.price * multiplier))
                id_str = f"🆔 {item_id} | " if item_id is not None else ""
                embed.add_field(
                    name=f"{id_str} {item.name}",
                    value=f"Vente: **${current_price}** (Base: ${item.price})",
                    inline=True
                )
                idx += 1
            await ctx.send(embed=embed)

    @commands.command(name='market_sell', aliases=["ms", "m_s"])
    async def sell(self, ctx, item_name: str, amount: int = 1):
        item_name = item_name.strip()
        if item_name.isdigit():
            resolved = get_item_name_by_id(int(item_name))
            if resolved:
                item_name = resolved
        
        user_id = ctx.author.id
        if item_name not in ITEMS_REGISTRY:
            await ctx.send("❌ Cet item n'existe pas !")
        if item_name not in self.sellable_items_names:
            await ctx.send("❌ Cet item ne peut pas être vendu sur le marché !")
        idx = self.sellable_items_names.index(item_name)
        final_price = max(1, int(ITEMS_REGISTRY[item_name].price * self.item_multipliers[idx]))
        total_gain = final_price * amount
        if has_item(user_id, item_name, amount):
            remove_item_from_inventory(user_id, item_name, amount)
            update_balance(user_id, total_gain)
            await ctx.send(f"💰 Tu as vendu **{amount}x {item_name}** pour **${total_gain}**.")
        else:
            await ctx.send(f"❌ Tu ne possèdes pas cet item **{item_name}**.")

async def setup(bot):
    await bot.add_cog(Market(bot))
