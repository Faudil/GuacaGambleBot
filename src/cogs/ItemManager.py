import discord
from discord.ui import Button, View
from discord.ext import commands

from src.command_decorators import daily_limit
from src.database.item import transfer_item_transaction, has_item, remove_item_from_inventory, get_item_name_by_id
from src.database.settings import get_language
from src.globals import ITEMS_REGISTRY
from src.utils.i18n import t, get_item_name


class TradeView(View):
    def __init__(self, seller, buyer, item_name, price, lang):
        super().__init__(timeout=60)
        self.seller = seller
        self.buyer = buyer
        self.item_name = item_name
        self.price = price
        self.lang = lang
        
        for item in self.children:
            if isinstance(item, Button):
                 if item.label == "✅ Accepter l'offre":
                     item.label = t("item_manager.accept_label", lang)
                 elif item.label == "Refuser":
                     item.label = t("item_manager.refuse_label", lang)

    @discord.ui.button(label="✅ Accepter l'offre", style=discord.ButtonStyle.success)
    async def confirm(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.buyer:
            return await interaction.response.send_message(t("item_manager.not_for_you", self.lang), ephemeral=True)

        result = transfer_item_transaction(self.seller.id, self.buyer.id, self.item_name, self.price)

        if result == "SUCCESS":
            display_name = get_item_name(self.item_name, self.lang)
            await interaction.response.edit_message(
                content=t("item_manager.trade_success", self.lang, seller=self.seller.mention, item=display_name, buyer=self.buyer.mention, price=self.price),
                view=None)
        elif result == "NO_MONEY":
            await interaction.response.send_message(t("item_manager.no_money", self.lang), ephemeral=True)
        elif result == "NO_ITEM":
            await interaction.response.send_message(t("item_manager.no_item", self.lang), ephemeral=True)
        else:
            await interaction.response.send_message(t("item_manager.unknown_error", self.lang), ephemeral=True)
        self.stop()
        return None

    @discord.ui.button(label="Refuser", style=discord.ButtonStyle.danger)
    async def cancel(self, interaction: discord.Interaction, button: Button):
        if interaction.user == self.buyer or interaction.user == self.seller:
            await interaction.response.edit_message(content=t("item_manager.trade_cancelled", self.lang), view=None)
            self.stop()


class ItemManager(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='use')
    @daily_limit("item", 5)
    async def use_item(self, ctx, item_name: str):
        """Utiliser un objet."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        item_name = item_name.strip()
        if item_name.isdigit():
            resolved = get_item_name_by_id(int(item_name))
            if resolved:
                item_name = resolved
        item_name = item_name.lower()
        display_name = get_item_name(item_name, lang)
        if not has_item(ctx.author.id, item_name):
            return await ctx.send(t("item_manager.no_item_owned", lang, item_name=display_name))
        if await ITEMS_REGISTRY[item_name].use(ctx):
            remove_item_from_inventory(ctx.author.id, item_name)
            return await ctx.send(t("item_manager.used_item", lang, item_name=display_name))
        return None

    @commands.command(name='sell')
    async def sell_item(self, ctx, recipient: discord.Member, item_name: str, price: int):
        """Vend un objet à un autre joueur."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        item_name = item_name.strip()
        if item_name.isdigit():
            resolved = get_item_name_by_id(int(item_name))
            if resolved:
                item_name = resolved
        if recipient.bot or recipient.id == ctx.author.id:
            return await ctx.send(t("economy.give_invalid", lang))
        if price < 0:
            return await ctx.send(t("loan.invalid_amount", lang))
        
        display_name = get_item_name(item_name, lang)
        embed = discord.Embed(title=t("item_manager.trade_proposal_title", lang), color=discord.Color.orange())
        embed.description = t("item_manager.trade_proposal_desc", lang, seller=ctx.author.mention, item=display_name, price=price, buyer=recipient.mention)

        view = TradeView(ctx.author, recipient, item_name, price, lang)
        return await ctx.send(content=recipient.mention, embed=embed, view=view)

async def setup(bot):
    await bot.add_cog(ItemManager(bot))
