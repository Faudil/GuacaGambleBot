import discord
from discord.ext import commands, tasks
import random
from discord.ui import Button, View

from src.command_decorators import daily_limit
from src.database.balance import get_balance, update_balance
from src.database.item import add_item_to_inventory, get_all_items_db

from src.globals import ITEMS_REGISTRY, ITEM_DROPPABLE
from src.database.settings import get_announcement_channel, get_language
from src.utils.i18n import t

from src.items.Beer import Beer
from src.items.Bow import Bow
from src.items.CheatCoin import CheatCoin
from src.items.Coffee import Coffee
from src.items.Fertilizer import Fertilizer
from src.items.ForgetPotion import ForgetPotion
from src.items.FortuneCookie import FortuneCookie
from src.items.Hook import Hook
from src.items.LandDeed import VegetablePatchDeed, GreenhouseDeed, OrchardDeed
from src.items.Magnet import Magnet, RustyMagnet, ElectricMagnet
from src.items.MysteryEgg import MysteryEgg
from src.items.ScratchTicket import ScratchTicket
from src.items.VipTicket import VipTicket
from src.items.LoreLog import DataDisk, OldJournal
from src.items.FarmItem import (
    WheatSeed, OatSeed, CornSeed, PotatoSeed, TomatoSeed,
    PumpkinSeed, CoffeeSeed, CocoaSeed, StrawberrySeed,
    GoldenAppleSeed, StarFruitSeed
)
from src.utils.NPCManager import NPCManager


class FlashSaleView(View):
    def __init__(self, item, price, lang):
        super().__init__(timeout=None)
        self.item = item
        self.price = price
        self.lang = lang
        self.buy.label = t("shop.buy_now_label", lang)

    @discord.ui.button(style=discord.ButtonStyle.green)
    async def buy(self, interaction: discord.Interaction, button: Button):
        bal = get_balance(interaction.user.id)
        if bal < self.price:
            return await interaction.response.send_message(t("shop.too_broke", self.lang), ephemeral=True)
            
        update_balance(interaction.user.id, -self.price)
        add_item_to_inventory(interaction.user.id, self.item.name)
        
        button.disabled = True
        button.label = t("shop.sold_label", self.lang)
        
        await interaction.response.send_message(
            t("shop.bought_msg", self.lang, user=interaction.user.display_name, item=self.item.display_name(self.lang)),
            ephemeral=False)
        self.stop()
        return await interaction.message.edit(view=self)


class DailyShopView(View):
    def __init__(self, user, offers, lang):
        super().__init__(timeout=120)
        self.user = user
        self.offers = offers
        self.lang = lang
        
        for idx, offer in enumerate(offers):
            item = offer['item']
            price = offer['price']
            is_discounted = offer['discounted']
            label_suffix = f" (${price})"
            if is_discounted:
                label_suffix += " 🔥"
            
            button = discord.ui.Button(
                label=f"{item.display_name(lang)}{label_suffix}",
                style=discord.ButtonStyle.green if is_discounted else discord.ButtonStyle.blurple,
                custom_id=f"buy_{idx}"
            )
            button.callback = self.make_callback(offer)
            self.add_item(button)

    def make_callback(self, offer):
        async def callback(interaction: discord.Interaction):
            if interaction.user.id != self.user.id:
                return await interaction.response.send_message(t("shop.not_your_shop", self.lang), ephemeral=True)
            
            item = offer['item']
            price = offer['price']
            
            bal = get_balance(interaction.user.id)
            if bal < price:
                return await interaction.response.send_message(t("shop.too_broke_item", self.lang), ephemeral=True)
                
            update_balance(interaction.user.id, -price)
            add_item_to_inventory(interaction.user.id, item.name)
            
            for child in self.children:
                child.disabled = True
            
            embed = interaction.message.embeds[0]
            embed.color = discord.Color.green()
            embed.set_footer(text=t("shop.bought_footer", self.lang, item=item.display_name(self.lang)))
            
            await interaction.response.send_message(
                t("shop.buy_success", self.lang, item=item.display_name(self.lang), price=price),
                ephemeral=False
            )
            await interaction.message.edit(embed=embed, view=self)
            self.stop()
        return callback


class Shop(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot
        self.drop_loop.start()

    def cog_unload(self):
        self.drop_loop.cancel()

    @tasks.loop(minutes=30)
    async def drop_loop(self):
        items = [Coffee(), VipTicket(), Beer(), ForgetPotion(),
                 Hook(), Fertilizer(), Bow(),
                 FortuneCookie(), CheatCoin(), Magnet(),
                 RustyMagnet(), ElectricMagnet(), ScratchTicket(),
                 VegetablePatchDeed(), GreenhouseDeed(), OrchardDeed(), MysteryEgg(),
                 DataDisk(), OldJournal(),
                 WheatSeed(), OatSeed(), CornSeed(), PotatoSeed(), TomatoSeed(),
                 PumpkinSeed(), CoffeeSeed(), CocoaSeed(), StrawberrySeed(),
                 GoldenAppleSeed(), StarFruitSeed()]
        if not items:
            return

        for guild in self.bot.guilds:
            lang = get_language(guild.id)
            if random.random() < 0.5:
                item = random.choice(items)
                discount = random.randint(30, 70) / 100
                price = max(1, int(item.price * (1 - discount)))
                
                channel_id = get_announcement_channel(guild.id)
                if channel_id == 0: continue
                channel = guild.get_channel(channel_id) if channel_id else guild.system_channel
                if not channel and guild.text_channels: channel = guild.text_channels[0]
                
                if channel:
                    # Flash sales are public, we don't apply individual NPC discounts here
                    # unless we want the announcer to be an NPC. Let's keep it simple for now.
                    embed = discord.Embed(title=t("shop.flash_sale_title", lang), color=discord.Color.gold())
                    embed.description = t("shop.flash_sale_desc", lang,
                                         item=item.display_name(lang),
                                         desc=item.display_description(lang),
                                         price=item.price,
                                         flash_price=price,
                                         discount=int(discount * 100))
                    embed.set_thumbnail(url="https://cdn-icons-png.flaticon.com/512/1170/1170679.png")
                    view = FlashSaleView(item, price, lang)
                    await channel.send(embed=embed, view=view)


    @commands.command(name='shop', aliases=['boutique'])
    @daily_limit("shop", 2)
    async def personal_shop(self, ctx):
        """Ouvre ta boutique personnelle avec 4 offres aléatoires (2 fois par jour)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        
        from src.database.housing import is_inventory_full
        full, current, limit = is_inventory_full(ctx.author.id)
        if full:
            return await ctx.send(t("housing.inv_full_warning", lang, current=current, limit=limit))
            
        items = [Coffee(), VipTicket(), Beer(), ForgetPotion(),
                 Hook(), Fertilizer(), Bow(),
                 FortuneCookie(), CheatCoin(), Magnet(),
                 RustyMagnet(), ElectricMagnet(), ScratchTicket(),
                 VegetablePatchDeed(), GreenhouseDeed(), OrchardDeed(), MysteryEgg(),
                 DataDisk(), OldJournal(),
                 WheatSeed(), OatSeed(), CornSeed(), PotatoSeed(), TomatoSeed(),
                 PumpkinSeed(), CoffeeSeed(), CocoaSeed(), StrawberrySeed(),
                 GoldenAppleSeed(), StarFruitSeed()]
                 
        selected_items = random.sample(items, min(len(items), 4))
        offers = []
        
        embed = discord.Embed(title=t("shop.personal_title", lang, user=ctx.author.display_name), color=discord.Color.blue())
        desc = t("shop.personal_desc", lang)
        
        bonuses = NPCManager.get_user_bonuses(ctx.author.id)
        npc_discount = bonuses.get("shop_discount", 0.0)
        
        for item in selected_items:
            is_discounted = random.random() < 0.35 # 35% chance for base discount
            base_price = item.price
            
            if is_discounted:
                random_discount = random.randint(5, 30) / 100
                price = max(1, int(base_price * (1 - random_discount)))
                # Combine with NPC discount multiplicatively? No, let's just take the best one or add them.
                # Let's say NPC discount is ADDED TO the existing one.
                final_discount = random_discount + npc_discount
                price = max(1, int(base_price * (1 - final_discount)))
                
                desc += f"**{item.display_name(lang)}** : ~~${base_price}~~ ➔ **${price}** 🔥 `(-{int(final_discount * 100)}%)`"
                if npc_discount > 0:
                    desc += f" (Bonus NPC: -{int(npc_discount * 100)}%)"
                desc += f"\n*{item.display_description(lang)}*\n\n"
            else:
                price = max(1, int(base_price * (1 - npc_discount)))
                if npc_discount > 0:
                    desc += f"**{item.display_name(lang)}** : ~~${base_price}~~ ➔ **${price}** 🎁 `(-{int(npc_discount * 100)}%)`\n"
                else:
                    desc += f"**{item.display_name(lang)}** : **${price}**\n"
                desc += f"*{item.display_description(lang)}*\n\n"
                
            offers.append({
                'item': item,
                'price': price,
                'discounted': is_discounted or npc_discount > 0
            })
        embed.description = desc
        embed.set_thumbnail(url=ctx.author.display_avatar.url)
        view = DailyShopView(ctx.author, offers, lang)
        message = await ctx.send(embed=embed, view=view)
        
        async def on_timeout():
            for child in view.children:
                child.disabled = True
            try:
                await message.edit(view=view)
            except:
                pass
        view.on_timeout = on_timeout


    @drop_loop.before_loop
    async def before_drop(self):
        await self.bot.wait_until_ready()


async def setup(bot):
    await bot.add_cog(Shop(bot))