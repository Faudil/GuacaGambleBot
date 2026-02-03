import discord
from discord.ui import Button, View
from discord.ext import commands

from src.data_handling import transfer_item_transaction, has_item
from src.globals import ITEMS_REGISTRY


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


class ItemManager(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='use')
    async def use_item(self, ctx, *, item_name: str):
        item_name = item_name.lower()
        if not has_item(ctx.author.id, item_name):
            return await ctx.send(f"❌ Tu n'as pas de **{item_name}**.")
        await ITEMS_REGISTRY[item_name].use(ctx)
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
    await bot.add_cog(ItemManager(bot))
