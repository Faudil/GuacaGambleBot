import discord
from discord.ext import commands, tasks
import random
from discord.ui import Button, View

from src.command_decorators import daily_limit
from src.database.balance import get_balance, update_balance
from src.database.item import add_item_to_inventory, get_all_items_db

from src.globals import CHANNEL_ID, ITEMS_REGISTRY, ITEM_DROPPABLE
from src.items.Beer import Beer
from src.items.CheatCoin import CheatCoin
from src.items.Coffee import Coffee
from src.items.FortuneCookie import FortuneCookie
from src.items.LandDeed import VegetablePatchDeed, GreenhouseDeed, OrchardDeed
from src.items.Magnet import Magnet, RustyMagnet, ElectricMagnet
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
        add_item_to_inventory(interaction.user.id, self.item['name'])
        self.stop()
        button.disabled = True
        button.label = "VENDU !"
        await interaction.response.send_message(
            f"✅ **{interaction.user.display_name}** a saisi l'offre ! Il obtient **{self.item['name']}**.",
            ephemeral=False)
        return await interaction.message.edit(view=self)



class Shop(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot
        self.drop_loop.start()

    def cog_unload(self):
        self.drop_loop.cancel()

    @tasks.loop(minutes=30)
    async def drop_loop(self):
        if random.random() < 0.7:
            items = [Coffee(), VipTicket(), Beer(),
                     FortuneCookie(), CheatCoin(), Magnet(),
                     RustyMagnet(), ElectricMagnet(), ScratchTicket(),
                     VegetablePatchDeed(), GreenhouseDeed(), OrchardDeed()]
            if not items: return

            item = random.choice(items)
            discount = random.randint(30, 70) / 100
            price = max(1, int(item['price'] * (1 - discount)))
            for guild in self.bot.guilds:
                channel = guild.get_channel(CHANNEL_ID)
                if channel:
                    embed = discord.Embed(title="⚡ VENTE FLASH !", color=discord.Color.gold())
                    embed.description = (
                        f"Un marchand itinérant propose : **{item['name']}**\n"
                        f"{item['description']}\n"
                        f"Prix normal : ~~${item['price']}~~\n"
                        f"**Prix Flash : ${price}** (-{discount * 100})%)"
                    )
                    embed.set_thumbnail(url="https://cdn-icons-png.flaticon.com/512/1170/1170679.png")
                    view = FlashSaleView(item, price)
                    await channel.send(embed=embed, view=view)


    @drop_loop.before_loop
    async def before_drop(self):
        await self.bot.wait_until_ready()


async def setup(bot):
    await bot.add_cog(Shop(bot))