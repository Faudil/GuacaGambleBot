import discord
from discord.ext import commands, tasks

from src.database.other import get_top_users, get_top_pets
from src.globals import CHANNEL_ID
from src.models.Pet import PETS_DB


class Leaderboard(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.current_top_pet_id = None
        self.top_elo_check_loop.start()

    def cog_unload(self):
        self.top_elo_check_loop.cancel()

    @tasks.loop(minutes=30)
    async def top_elo_check_loop(self):
        top_pets = get_top_pets(1)
        if not top_pets:
            return

        current_top = top_pets[0]
        new_top_pet_id = current_top["pet_id"]

        if self.current_top_pet_id is None:
            self.current_top_pet_id = new_top_pet_id
            return

        if new_top_pet_id != self.current_top_pet_id:
            self.current_top_pet_id = new_top_pet_id

            for guild in self.bot.guilds:
                channel = guild.get_channel(CHANNEL_ID)
                if not channel and guild.text_channels:
                    channel = guild.text_channels[0]
                if channel:
                    member = guild.get_member(int(current_top["user_id"]))
                    owner_name = member.display_name if member else "Inconnu"
                    
                    pet_info = PETS_DB.get(current_top["pet_type"], {})
                    pet_emoji = pet_info.get("emoji", "🐾")
                    pet_name = current_top["nickname"] or current_top["pet_type"]

                    embed = discord.Embed(
                        title="🌟 NOUVEAU #1 CLASSEMENT ELO ! 🌟",
                        description=f"Le familier **{pet_name}** {pet_emoji} de **{owner_name}** vient de prendre la première place du classement mondial des familiers avec **{current_top['elo']} ELO** !",
                        color=discord.Color.gold()
                    )
                    await channel.send(embed=embed)
                    break

    @top_elo_check_loop.before_loop
    async def before_top_elo_check_loop(self):
        await self.bot.wait_until_ready()

    @commands.command(name='richest', aliases=['top_wealth', 'classement_richesse'])
    async def leaderboard(self, ctx):
        """Display the 5 richest in the server."""
        users = list(get_top_users(5))
        if not users:
            return await ctx.send("Personne n'a d'argent pour l'instant !")
        sorted_users = sorted(users, key=lambda x: x["balance"], reverse=True)
        embed = discord.Embed(title="🏆 Classement des plus riches", color=discord.Color.gold())
        description = ""
        for i, user in enumerate(sorted_users, 1):
            member = ctx.guild.get_member(int(user["user_id"]))
            name = member.display_name if member else "Inconnu"
            medals = {1: "🥇", 2: "🥈", 3: "🥉"}
            rank_emoji = medals.get(i, "🔹")
            description += f"{rank_emoji} **{name}** : ${user['balance']}\n"
        embed.description = description
        return await ctx.send(embed=embed)

    @commands.command(name='top', aliases=['leaderboard', 'classement_gloire'])
    async def glory_leaderboard(self, ctx):
        """Affiche le classement de Gloire."""
        from src.database.other import get_top_glory_users
        users = get_top_glory_users(5)
        if not users:
            return await ctx.send("Personne n'a de points de gloire pour l'instant !")
        
        embed = discord.Embed(title="🌟 Classement de Gloire", color=discord.Color.gold())
        description = ""
        for i, user in enumerate(users, 1):
            member = ctx.guild.get_member(int(user["user_id"]))
            name = member.display_name if member else "Inconnu"
            medals = {1: "🥇", 2: "🥈", 3: "🥉"}
            rank_emoji = medals.get(i, "🔹")
            description += f"{rank_emoji} **{name}** : {user['glory']} pts de Gloire\n"
            
        embed.description = description
        return await ctx.send(embed=embed)

    @commands.command(name='top_pets', aliases=['pet_leaderboard', 'classement_pets'])
    async def pet_leaderboard(self, ctx):
        """Affiche le classement ELO des familiers."""
        from src.database.other import get_top_pets
        from src.models.Pet import PETS_DB
        pets = get_top_pets(5)
        if not pets:
            return await ctx.send("Personne n'a de familier classé pour l'instant !")
        
        embed = discord.Embed(title="🐾 Classement des Familiers (ELO)", color=discord.Color.green())
        description = ""
        for i, pet in enumerate(pets, 1):
            member = ctx.guild.get_member(int(pet["user_id"]))
            owner_name = member.display_name if member else "Inconnu"
            medals = {1: "🥇", 2: "🥈", 3: "🥉"}
            rank_emoji = medals.get(i, "🔹")
            
            pet_info = PETS_DB.get(pet["pet_type"], {})
            pet_emoji = pet_info.get("emoji", "🐾")
            pet_name = pet["nickname"] or pet["pet_type"]
            
            description += f"{rank_emoji} **{pet_name}** {pet_emoji} ({owner_name}) : {pet['elo']} ELO\n"
            
        embed.description = description
        return await ctx.send(embed=embed)

async def setup(bot):
    await bot.add_cog(Leaderboard(bot))
