import discord
from discord.ext import commands

from src.database.item import get_all_user_inventory
from src.database.settings import get_language
from src.globals import ITEMS_REGISTRY
from src.items.Item import ItemRarity
from src.utils.i18n import t, get_rarity_name, get_item_name


class Inventory(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    def get_rarity_emoji(self, rarity: ItemRarity) -> str:
        emojis = {
            ItemRarity.common: "⚪",
            ItemRarity.rare: "🟢",
            ItemRarity.epic: "🔵",
            ItemRarity.legendary: "🟣",
            ItemRarity.unique: "⭐"
        }
        return emojis.get(rarity, "⚪")

    @commands.command(name='inventory', aliases=['inv', 'bag', 'sac'])
    async def inventory(self, ctx, user: discord.Member = None):
        """Voir ton sac à dos et tes objets."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user = ctx.author if user is None else user
        
        from src.database.housing import is_inventory_full
        full, current, limit = is_inventory_full(user.id)
        
        inventory = get_all_user_inventory(user.id)
        if not inventory:
            return await ctx.send(t("inventory.empty", lang, user=user.mention) + f"\n📦 **Slots: `{current}/{limit}`**")

        embed = discord.Embed(
            title=t("inventory.title", lang, user=user.name) + f" ({current}/{limit})",
            color=discord.Color.blue()
        )
        description_lines = []
        for item in inventory:
            obj_name = item['name']
            quantity = item['quantity']
            if quantity == 0:
                continue
            item_id = item['id']
            obj = ITEMS_REGISTRY[obj_name]
            emoji = self.get_rarity_emoji(obj.rarity)
            display_name = get_item_name(obj_name, lang)
            line = f"🆔 `{item_id}` | {emoji} **{display_name}** : `x{quantity}`"
            description_lines.append(line)
        full_text = "\n".join(description_lines)
        embed.description = full_text
        embed.set_footer(text=t("inventory.footer", lang))
        return await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Inventory(bot))