import discord
from discord.ext import commands, tasks
import random
from discord.ui import Button, View

from src.data_handling import (
    get_all_items_db, add_item_to_inventory, use_item_db,
    transfer_item_transaction, update_balance, get_balance, has_item
)
from src.globals import CHANNEL_ID, ITEMS_REGISTRY


class FlashSaleView(View):
    def __init__(self, item, price):
        super().__init__(timeout=300)
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


class TradeView(View):
    def __init__(self, seller, buyer, item_name, price):
        super().__init__(timeout=60)
        self.seller = seller
        self.buyer = buyer
        self.item_name = item_name
        self.price = price

    @discord.ui.button(label="✅ Accepter l'offre", style=discord.ButtonStyle.success)
    async def confirm(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.buyer:
            return await interaction.response.send_message("Ce n'est pas pour toi !", ephemeral=True)

        result = transfer_item_transaction(self.seller.id, self.buyer.id, self.item_name, self.price)

        if result == "SUCCESS":
            await interaction.response.edit_message(
                content=f"🤝 **Affaire conclue !**\n{self.seller.mention} a vendu **{self.item_name}** à {self.buyer.mention} pour **${self.price}**.",
                view=None
            )
        elif result == "NO_MONEY":
            await interaction.response.send_message("❌ Tu n'as pas assez d'argent.", ephemeral=True)
        elif result == "NO_ITEM":
            await interaction.response.send_message("❌ Le vendeur n'a plus l'objet !", ephemeral=True)
        else:
            await interaction.response.send_message("❌ Erreur inconnue.", ephemeral=True)
        self.stop()
        return None

    @discord.ui.button(label="Refuser", style=discord.ButtonStyle.danger)
    async def cancel(self, interaction: discord.Interaction, button: Button):
        if interaction.user == self.buyer or interaction.user == self.seller:
            await interaction.response.edit_message(content="❌ Échange annulé.", view=None)
            self.stop()


class Shop(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot
        self.drop_loop.start()

    def cog_unload(self):
        self.drop_loop.cancel()

    @tasks.loop(minutes=45)
    async def drop_loop(self):
        if random.random() < 0.5:
            items = get_all_items_db()
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
                        f"Prix normal : ~~${item['price']}~~\n"
                        f"**Prix Flash : ${price}** (-{discount})%)"
                    )
                    embed.set_thumbnail(url="https://cdn-icons-png.flaticon.com/512/1170/1170679.png")
                    view = FlashSaleView(item, price)
                    await channel.send(embed=embed, view=view)

    @drop_loop.before_loop
    async def before_drop(self):
        await self.bot.wait_until_ready()

    @commands.command(name='use')
    async def use_item(self, ctx, *, item_name: str):
        if not has_item(ctx.author.id, item_name):
            return await ctx.send(f"❌ Tu n'as pas de **{item_name}**.")
        name_clean = item_name.lower()
        await ITEMS_REGISTRY[name_clean].use(ctx)
        msg = f"✨ Tu as utilisé **{item_name}**..."
        return await ctx.send(msg)

    @commands.command(name='sell')
    async def sell_item(self, ctx, recipient: discord.Member, price: int, item_name: str):
        if recipient.bot or recipient.id == ctx.author.id:
            return await ctx.send("❌ Transaction impossible.")
        if price < 0:
            return await ctx.send("❌ Prix invalide.")
        embed = discord.Embed(title="🤝 Proposition d'échange", color=discord.Color.orange())
        embed.description = (
            f"**{ctx.author.mention}** veut te vendre **{item_name}** pour **${price}**.\n"
            f"{recipient.mention}, acceptes-tu ?"
        )

        view = TradeView(ctx.author, recipient, item_name, price)
        return await ctx.send(content=recipient.mention, embed=embed, view=view)


async def setup(bot):
    await bot.add_cog(Shop(bot))