import discord
from discord.ext import commands, tasks
import random
from discord.ui import Button, View

from src.command_decorators import daily_limit
from src.database.balance import get_balance, update_balance
from src.database.item import add_item_to_inventory, get_all_items_db

from src.globals import CHANNEL_ID, ITEMS_REGISTRY, ITEM_DROPPABLE
from src.items.Beer import Beer
from src.items.Bow import Bow
from src.items.CheatCoin import CheatCoin
from src.items.Coffee import Coffee
from src.items.Fertilizer import Fertilizer
from src.items.FortuneCookie import FortuneCookie
from src.items.Hook import Hook
from src.items.LandDeed import VegetablePatchDeed, GreenhouseDeed, OrchardDeed
from src.items.Magnet import Magnet, RustyMagnet, ElectricMagnet
from src.items.MysteryEgg import MysteryEgg
from src.items.ScratchTicket import ScratchTicket
from src.items.VipTicket import VipTicket


class FlashSaleView(View):
    def __init__(self, item, price):
        super().__init__(timeout=None)
        self.item = item
        self.price = price

    @discord.ui.button(label="🛒 ACHETER MAINTENANT !", style=discord.ButtonStyle.green)
    async def buy(self, interaction: discord.Interaction, button: Button):
        bal = get_balance(interaction.user.id)
        if bal < self.price:
            return await interaction.response.send_message("❌ Tu es trop pauvre !", ephemeral=True)
        update_balance(interaction.user.id, -self.price)
        add_item_to_inventory(interaction.user.id, self.item.name)
        self.stop()
        button.disabled = True
        button.label = "VENDU !"
        await interaction.response.send_message(
            f"✅ **{interaction.user.display_name}** a saisi l'offre ! Il obtient **{self.item.name}**.",
            ephemeral=False)
        return await interaction.message.edit(view=self)


class DailyShopView(View):
    def __init__(self, user, offers):
        super().__init__(timeout=120)
        self.user = user
        self.offers = offers
        
        for idx, offer in enumerate(offers):
            item = offer['item']
            price = offer['price']
            is_discounted = offer['discounted']
            label_suffix = f" (${price})"
            if is_discounted:
                label_suffix += " 🔥"
            
            button = discord.ui.Button(
                label=f"{item.name}{label_suffix}",
                style=discord.ButtonStyle.green if is_discounted else discord.ButtonStyle.blurple,
                custom_id=f"buy_{idx}"
            )
            button.callback = self.make_callback(offer)
            self.add_item(button)

    def make_callback(self, offer):
        async def callback(interaction: discord.Interaction):
            if interaction.user.id != self.user.id:
                return await interaction.response.send_message("❌ Ce n'est pas ta boutique !", ephemeral=True)
            
            item = offer['item']
            price = offer['price']
            
            bal = get_balance(interaction.user.id)
            if bal < price:
                return await interaction.response.send_message("❌ Tu es trop pauvre pour cet article !", ephemeral=True)
                
            update_balance(interaction.user.id, -price)
            add_item_to_inventory(interaction.user.id, item.name)
            
            for child in self.children:
                child.disabled = True
            
            embed = interaction.message.embeds[0]
            embed.color = discord.Color.green()
            embed.set_footer(text=f"Acheté : {item.name}")
            
            await interaction.response.send_message(
                f"✅ Tu as acheté **{item.name}** pour **${price}** !",
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
        if random.random() < 0.5:
            items = [Coffee(), VipTicket(), Beer(),
                     Hook(), Fertilizer(), Bow(),
                     FortuneCookie(), CheatCoin(), Magnet(),
                     RustyMagnet(), ElectricMagnet(), ScratchTicket(),
                     VegetablePatchDeed(), GreenhouseDeed(), OrchardDeed(), MysteryEgg()]
            if not items:
                return
            item = random.choice(items)
            discount = random.randint(30, 70) / 100
            price = max(1, int(item.price * (1 - discount)))
            for guild in self.bot.guilds:
                channel = guild.get_channel(CHANNEL_ID)
                if channel:
                    embed = discord.Embed(title="⚡ VENTE FLASH !", color=discord.Color.gold())
                    embed.description = (
                        f"Un marchand itinérant propose : **{item.name}**\n"
                        f"{item.description}\n"
                        f"Prix normal : ~~${item.price}~~\n"
                        f"**Prix Flash : ${price}** (-{discount * 100})%)"
                    )
                    embed.set_thumbnail(url="https://cdn-icons-png.flaticon.com/512/1170/1170679.png")
                    view = FlashSaleView(item, price)
                    await channel.send(embed=embed, view=view)


    @commands.command(name='shop', aliases=['boutique'])
    @daily_limit("shop", 2)
    async def personal_shop(self, ctx):
        """Ouvre ta boutique personnelle avec 4 offres aléatoires (2 fois par jour)."""
        items = [Coffee(), VipTicket(), Beer(),
                 Hook(), Fertilizer(), Bow(),
                 FortuneCookie(), CheatCoin(), Magnet(),
                 RustyMagnet(), ElectricMagnet(), ScratchTicket(),
                 VegetablePatchDeed(), GreenhouseDeed(), OrchardDeed(), MysteryEgg()]
                 
        selected_items = random.sample(items, 4)
        offers = []
        
        embed = discord.Embed(title=f"🛒 Boutique de {ctx.author.display_name}", color=discord.Color.blue())
        embed.description = "Choisis un article à acheter ! Attention, **tu ne peux acheter qu'un seul article** parmi les 4.\n\n"
        
        for item in selected_items:
            is_discounted = random.random() < 0.35 # 35% chance for discount
            if is_discounted:
                discount = random.randint(5, 30) / 100
                price = max(1, int(item.price * (1 - discount)))
                embed.description += f"**{item.name}** : ~~${item.price}~~ ➔ **${price}** 🔥 `(-{int(discount * 100)}%)`\n"
                embed.description += f"*{item.description}*\n\n"
            else:
                price = item.price
                embed.description += f"**{item.name}** : **${price}**\n"
                embed.description += f"*{item.description}*\n\n"
                
            offers.append({
                'item': item,
                'price': price,
                'discounted': is_discounted
            })
        embed.set_thumbnail(url=ctx.author.display_avatar.url)
        view = DailyShopView(ctx.author, offers)
        message = await ctx.send(embed=embed, view=view)
        
        async def on_timeout():
            for child in view.children:
                child.disabled = True
            await message.edit(view=view)
        view.on_timeout = on_timeout


    @drop_loop.before_loop
    async def before_drop(self):
        await self.bot.wait_until_ready()


async def setup(bot):
    await bot.add_cog(Shop(bot))