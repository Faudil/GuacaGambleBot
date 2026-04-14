import itertools

import discord
from discord.ext import commands, tasks
import random

from src.database.achievement import check_and_unlock_achievements, format_achievements_unlocks
from src.database.balance import update_balance
from src.database.item import remove_item_from_inventory, get_item_name_by_id, get_item_id_from_name, \
    has_item
from src.globals import ITEMS_REGISTRY
from src.items.FarmItem import Wheat, Oat, Corn, Tomato, Pumpkin, Potato, CoffeeBean, CocoaBean, Strawberry, \
    GoldenApple, StarFruit, RottenPlant
from src.items.MiningLoot import Coal, IronOre, GoldNugget, Diamond, Pebble, SilverOre, CopperOre, Emerald, \
    PlatinumOre
from src.items.FishingLoot import OldBoot, Trout, Salmon, Pufferfish, Swordfish, Sardine, KrakenTentacle, Carp, Whale, \
    Shark
from src.database.settings import get_language
from src.utils.i18n import t, get_item_name


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
        
        self.total_items_nbr = len(self.mining_items) + len(self.fishing_items) + len(self.farming_items)
        self.sellable_items = [self.mining_items, self.fishing_items, self.farming_items]
        self.sellable_items_names = [item.name.lower() for item in itertools.chain(*self.sellable_items)]
        self.item_multipliers = [1] * self.total_items_nbr
        self.bot = bot
        self.guild_multipliers = {}
        self.trend = "stable"
        self.update_market_prices.start()

    def cog_unload(self):
        self.update_market_prices.cancel()

    @tasks.loop(minutes=5)
    async def update_market_prices(self):
        for guild in self.bot.guilds:
            if guild.id not in self.guild_multipliers:
                self.guild_multipliers[guild.id] = [1] * self.total_items_nbr
                
            multipliers = self.guild_multipliers[guild.id]
            for i in range(0, len(multipliers)):
                change = random.choice([-0.1, -0.05, 0, 0.05, 0.1])
                multiplier = multipliers[i]
                multiplier += change
                multiplier = max(0.1, min(3, multiplier))
                multipliers[i] = multiplier

    @commands.command(name='market')
    async def show_market(self, ctx, to_show: str = None):
        """Voir le cours de la bourse (Krach ou Boom ?)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        mine_tag = ["mine", "minage", "mining"]
        fish_tag = ["fish", "pêche", "fishing"]
        farm_tag = ["farm", "ferme", "farming"]
        
        to_show_items = self.sellable_items
        to_show_keys = ["title_mining", "title_fishing", "title_farming"]
        
        start_indices = [0, len(self.mining_items), len(self.mining_items) + len(self.fishing_items)]
        
        selected_indices = [0, 1, 2]
        if to_show in mine_tag:
            selected_indices = [0]
        elif to_show in fish_tag:
            selected_indices = [1]
        elif to_show in farm_tag:
            selected_indices = [2]

        guild_id = ctx.guild.id if ctx.guild else None
        multipliers = self.guild_multipliers.get(guild_id, [1] * self.total_items_nbr) if guild_id else [1] * self.total_items_nbr

        for i in selected_indices:
            items = to_show_items[i]
            title_key = to_show_keys[i]
            idx = start_indices[i]
            
            embed = discord.Embed(title=t(f"market.{title_key}", lang), color=discord.Color.gold())
            for item in items:
                item_id = get_item_id_from_name(item.name)
                multiplier = multipliers[idx]
                current_price = int(max(1, item.price * multiplier))
                id_str = f"🆔 {item_id} | " if item_id is not None else ""
                
                embed.add_field(
                    name=f"{id_str} {get_item_name(item.name, lang)}",
                    value=t("market.sale_price", lang, price=current_price, base=item.price),
                    inline=True
                )
                idx += 1
            await ctx.send(embed=embed)

    @commands.command(name='market_sell', aliases=["ms", "m_s"])
    async def sell(self, ctx, item_name: str, amount: int = 1):
        """Vendre tes ressources au prix du marché."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        item_name = item_name.strip()
        if item_name.isdigit():
            resolved = get_item_name_by_id(int(item_name))
            if resolved:
                item_name = resolved
            else:
                return await ctx.send(t("market.invalid_id", lang))
        
        user_id = ctx.author.id
        if item_name not in ITEMS_REGISTRY:
            return await ctx.send(t("market.item_not_found", lang))
            
        if item_name.lower() not in self.sellable_items_names:
            return await ctx.send(t("market.item_not_sellable", lang))
            
        idx = self.sellable_items_names.index(item_name.lower())
        guild_id = ctx.guild.id if ctx.guild else None
        multipliers = self.guild_multipliers.get(guild_id, [1] * self.total_items_nbr) if guild_id else [1] * self.total_items_nbr
        final_price = max(1, int(ITEMS_REGISTRY[item_name].price * multipliers[idx]))
        total_gain = final_price * amount
        
        if has_item(user_id, item_name, amount):
            remove_item_from_inventory(user_id, item_name, amount)
            update_balance(user_id, total_gain)
            await ctx.send(t("market.sold_msg", lang, amount=amount, item=get_item_name(item_name, lang), gain=total_gain))
        else:
            await ctx.send(t("market.no_item", lang, amount=amount, item=get_item_name(item_name, lang)))

        from src.database.achievement import increment_stat
        increment_stat(user_id, "items_sold_market", amount)

        unlocks = check_and_unlock_achievements(int(user_id))
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks, lang))

async def setup(bot):
    await bot.add_cog(Market(bot))
