import discord
from discord.ext import commands

from src.database.balance import update_balance
from src.database.other import add_money_to_all
from src.database.item import add_item_to_inventory, get_item_name_by_id, add_item_to_all


class Admin(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='airdrop', aliases=['rain'])
    @commands.has_permissions(administrator=True)
    async def airdrop(self, ctx, user: discord.Member, amount: int):
        """Donner de l'argent à un joueur (Admin)."""
        if amount <= 0:
            return await ctx.send("❌ Le montant doit être positif.")
        update_balance(user.id, amount)
        embed = discord.Embed(
            title="💸 PLUIE DE BILLETS ! 💸",
            description=f"Je donnne **${amount}** à mon préféré {user.display_name} !",
            color=discord.Color.gold()
        )
        embed.set_footer(text=f"Offert par {ctx.author.display_name}")
        return await ctx.send(embed=embed)

    @airdrop.error
    async def airdrop_error(self, ctx, error):
        if isinstance(error, commands.MissingPermissions):
            await ctx.send("⛔ Tu n'as pas la permission de faire pleuvoir l'argent !")

    @commands.command('airdrop_item', aliases=['rain_item'])
    async def airdrop_item(self, ctx, user: discord.Member, item_name: str):
        """Donner un objet à un joueur (Admin)."""
        add_item_to_inventory(user.id, item_name.lower())
        embed = discord.Embed(
            title="💸 PLUIE D'OBJETS ! 💸",
            description=f"Je donnne **${item_name}** à mon préféré {user.display_name} !",
            color=discord.Color.gold()
        )
        embed.set_footer(text=f"Offert par {ctx.author.display_name}")
        return await ctx.send(embed=embed)

    @commands.command('airdrop_all', aliases=['rain_all'])
    @commands.has_permissions(administrator=True)
    async def airdrop_all(self, ctx, amount: int):
        """Donner de l'argent à tous les joueurs (Admin)."""
        if amount <= 0:
            return await ctx.send("❌ Le montant doit être positif.")
        
        rows = add_money_to_all(amount)
        embed = discord.Embed(
            title="💸 PLUIE DE BILLETS POUR TOUT LE MONDE ! 💸",
            description=f"Je donne **${amount}** à **{rows}** joueurs !",
            color=discord.Color.gold()
        )
        embed.set_footer(text=f"Offert par {ctx.author.display_name}")
        return await ctx.send(embed=embed)

    @commands.command('airdrop_item_all', aliases=['rain_item_all'])
    @commands.has_permissions(administrator=True)
    async def airdrop_item_all(self, ctx, item_name: str, quantity: int = 1):
        """Donner un objet à tous les joueurs (Admin)."""
        item_name = item_name.strip()
        if item_name.isdigit():
            resolved = get_item_name_by_id(int(item_name))
            if resolved:
                item_name = resolved
        if quantity <= 0:
            return await ctx.send("❌ La quantité doit être positive.")
        
        rows = add_item_to_all(item_name.lower(), quantity)
        if rows == 0:
            return await ctx.send("❌ Cet objet n'existe pas ou il n'y a aucun joueur en base.")
            
        embed = discord.Embed(
            title="💸 PLUIE D'OBJETS POUR TOUT LE MONDE ! 💸",
            description=f"Je donne **{quantity}x {item_name}** à **{rows}** joueurs !",
            color=discord.Color.gold()
        )
        embed.set_footer(text=f"Offert par {ctx.author.display_name}")
        return await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Admin(bot))
