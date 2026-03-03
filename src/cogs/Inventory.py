import discord
from discord.ext import commands

from src.database.item import get_all_user_inventory
from src.globals import ITEMS_REGISTRY
from src.items.Item import ItemRarity


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
        user = ctx.author if user is None else user
        inventory = get_all_user_inventory(user.id)
        if not inventory:
            return await ctx.send(
                f"🎒 Ton sac est vide, {user.mention} ! Va miner ou achète des trucs.")

        embed = discord.Embed(
            title=f"🎒 Sac à dos de {user.name}",
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
            line = f"🆔 `{item_id}` | {emoji} **{obj_name}** : `x{quantity}`"
            description_lines.append(line)
        full_text = "\n".join(description_lines)
        embed.description = full_text
        embed.set_footer(text="Utilise !use <nom de l'objet> pour t'en servir !")
        return await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Inventory(bot))