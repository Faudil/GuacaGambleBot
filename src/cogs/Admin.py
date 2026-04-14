import discord
from discord.ext import commands

from src.database.balance import update_balance
from src.database.other import add_money_to_all
from src.database.item import add_item_to_inventory, get_item_name_by_id, add_item_to_all
from src.database.settings import set_announcement_channel, disable_announcements, get_language, set_language
from src.utils.i18n import t


class Admin(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='airdrop', aliases=['rain'])
    @commands.has_permissions(administrator=True)
    async def airdrop(self, ctx, user: discord.Member, amount: int):
        """Donner de l'argent à un joueur (Admin)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if amount <= 0:
            return await ctx.send(t("admin.amount_positive", lang))
        update_balance(user.id, amount)
        embed = discord.Embed(
            title=t("admin.airdrop_title", lang),
            description=t("admin.airdrop_desc", lang, amount=amount, user=user.display_name),
            color=discord.Color.gold()
        )
        embed.set_footer(text=t("admin.gifted_by", lang, author=ctx.author.display_name))
        return await ctx.send(embed=embed)

    @airdrop.error
    async def airdrop_error(self, ctx, error):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if isinstance(error, commands.MissingPermissions):
            await ctx.send(t("admin.no_permission_money", lang))

    @commands.command('airdrop_item', aliases=['rain_item'])
    async def airdrop_item(self, ctx, user: discord.Member, item_name: str):
        """Donner un objet à un joueur (Admin)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        add_item_to_inventory(user.id, item_name.lower())
        embed = discord.Embed(
            title=t("admin.airdrop_item_title", lang),
            description=t("admin.airdrop_item_desc", lang, item_name=item_name, user=user.display_name),
            color=discord.Color.gold()
        )
        embed.set_footer(text=t("admin.gifted_by", lang, author=ctx.author.display_name))
        return await ctx.send(embed=embed)

    @commands.command('airdrop_all', aliases=['rain_all'])
    @commands.has_permissions(administrator=True)
    async def airdrop_all(self, ctx, amount: int):
        """Donner de l'argent à tous les joueurs (Admin)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if amount <= 0:
            return await ctx.send(t("admin.amount_positive", lang))
        
        rows = add_money_to_all(amount)
        embed = discord.Embed(
            title=t("admin.airdrop_all_title", lang),
            description=t("admin.airdrop_all_desc", lang, amount=amount, rows=rows),
            color=discord.Color.gold()
        )
        embed.set_footer(text=t("admin.gifted_by", lang, author=ctx.author.display_name))
        return await ctx.send(embed=embed)

    @commands.command('airdrop_item_all', aliases=['rain_item_all'])
    @commands.has_permissions(administrator=True)
    async def airdrop_item_all(self, ctx, item_name: str, quantity: int = 1):
        """Donner un objet à tous les joueurs (Admin)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        item_name = item_name.strip()
        if item_name.isdigit():
            resolved = get_item_name_by_id(int(item_name))
            if resolved:
                item_name = resolved
        if quantity <= 0:
            return await ctx.send(t("admin.quantity_positive", lang))
        
        rows = add_item_to_all(item_name.lower(), quantity)
        if rows == 0:
            return await ctx.send(t("admin.item_not_found", lang))
            
        embed = discord.Embed(
            title=t("admin.airdrop_item_all_title", lang),
            description=t("admin.airdrop_item_all_desc", lang, quantity=quantity, item_name=item_name, rows=rows),
            color=discord.Color.gold()
        )
        embed.set_footer(text=t("admin.gifted_by", lang, author=ctx.author.display_name))
        return await ctx.send(embed=embed)

    @commands.command(name='set_channel', aliases=["setchannel", "annonce_ici"])
    @commands.has_permissions(administrator=True)
    async def set_channel(self, ctx, channel: discord.TextChannel = None):
        """Définit le salon d'annonce pour ce serveur. (Admin uniquement)"""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        target_channel = channel or ctx.channel
        set_announcement_channel(ctx.guild.id, target_channel.id)
        await ctx.send(t("admin.channel_set", lang, channel=target_channel.mention))

    @commands.command(name='disable_announcements', aliases=["stop_annonces"])
    @commands.has_permissions(administrator=True)
    async def disable_announcements_cmd(self, ctx):
        """Désactive l'envoi d'annonces automatiques du bot sur ce serveur. (Admin uniquement)"""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        disable_announcements(ctx.guild.id)
        await ctx.send(t("admin.announcements_disabled", lang))

    @commands.command(name='setlang', aliases=["lang"])
    @commands.has_permissions(administrator=True)
    async def setlang_cmd(self, ctx, lang: str):
        """Choisi la langue du bot pour ce serveur. (Admin uniquement)"""
        current_lang = get_language(ctx.guild.id if ctx.guild else None)
        if lang.lower() not in ["fr", "en"]:
            return await ctx.send(t("admin.lang_not_supported", current_lang))
        set_language(ctx.guild.id, lang.lower())
        await ctx.send(t("admin.lang_set", lang.lower()))

    @commands.command(name='givecrowns', aliases=['addcrowns'])
    @commands.has_permissions(administrator=True)
    async def givecrowns(self, ctx, user: discord.Member, amount: int):
        """Donner des Crowns à un joueur (Admin)."""
        from src.database.housing import update_crowns
        update_crowns(user.id, amount)
        await ctx.send(f"✅ Added `{amount}` 👑 Crowns to {user.display_name}.")


async def setup(bot):
    await bot.add_cog(Admin(bot))
