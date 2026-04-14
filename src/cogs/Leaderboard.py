import discord
from discord.ext import commands, tasks

from src.database.other import get_top_users, get_top_pets
from src.database.settings import get_announcement_channel, get_language
from src.models.Pet import PETS_DB
from src.utils.i18n import t, get_pet_name


class Leaderboard(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.current_top_pet_id = None
        self.top_elo_check_loop.start()

    def cog_unload(self):
        self.top_elo_check_loop.cancel()

    @tasks.loop(minutes=30)
    async def top_elo_check_loop(self):
        for guild in self.bot.guilds:
            lang = get_language(guild.id)
            top_pets = get_top_pets(5, server_id=guild.id)
            if not top_pets:
                continue

            current_top = top_pets[0]
            new_top_pet_id = current_top["pet_id"]
            
            top_5_user_ids = {int(p["user_id"]) for p in top_pets}
            role_name = t("leaderboard.role_name", lang)
            role = discord.utils.get(guild.roles, name=role_name)
            if not role:
                try:
                    role = await guild.create_role(name=role_name, color=discord.Color.gold(), hoist=True, reason=t("leaderboard.role_reason", lang))
                except discord.Forbidden:
                    pass
            
            if role:
                for member in role.members:
                    if member.id not in top_5_user_ids:
                        try:
                            await member.remove_roles(role, reason=t("leaderboard.role_removed", lang))
                        except discord.Forbidden:
                            pass
                
                for user_id in top_5_user_ids:
                    member = guild.get_member(user_id)
                    if member and role not in member.roles:
                        try:
                            await member.add_roles(role, reason=t("leaderboard.role_added", lang))
                        except discord.Forbidden:
                            pass

            guild_top_pet_id = getattr(self, f"current_top_pet_id_{guild.id}", None)

            if guild_top_pet_id is None:
                setattr(self, f"current_top_pet_id_{guild.id}", new_top_pet_id)
                continue

            if new_top_pet_id != guild_top_pet_id:
                setattr(self, f"current_top_pet_id_{guild.id}", new_top_pet_id)

                channel_id = get_announcement_channel(guild.id)
                if channel_id == 0: continue
                channel = guild.get_channel(channel_id) if channel_id else guild.system_channel
                if not channel and guild.text_channels: channel = guild.text_channels[0]
                
                if channel:
                    member = guild.get_member(int(current_top["user_id"]))
                    owner_name = member.display_name if member else t("leaderboard.unknown", lang)
                    
                    pet_info = PETS_DB.get(current_top["pet_type"], {})
                    pet_emoji = pet_info.get("emoji", "🐾")
                    pet_name = current_top["nickname"] or get_pet_name(current_top["pet_type"], lang)

                    embed = discord.Embed(
                        title=t("leaderboard.new_top_pet_title", lang),
                        description=t("leaderboard.new_top_pet_desc", lang, pet_name=pet_name, pet_emoji=pet_emoji, owner_name=owner_name, elo=current_top['elo']),
                        color=discord.Color.gold()
                    )
                    await channel.send(embed=embed)

    @top_elo_check_loop.before_loop
    async def before_top_elo_check_loop(self):
        await self.bot.wait_until_ready()

    @commands.command(name='richest', aliases=['top_wealth', 'classement_richesse'])
    async def leaderboard(self, ctx):
        """Display the 5 richest in the server."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        server_member_ids = [member.id for member in ctx.guild.members]
        users = list(get_top_users(5, server_member_ids=server_member_ids))
        if not users:
            return await ctx.send(t("leaderboard.no_money", lang))
        sorted_users = sorted(users, key=lambda x: x["balance"], reverse=True)
        embed = discord.Embed(title=t("leaderboard.wealth_title", lang), color=discord.Color.gold())
        description = ""
        for i, user in enumerate(sorted_users, 1):
            member = ctx.guild.get_member(int(user["user_id"]))
            name = member.display_name if member else t("leaderboard.unknown", lang)
            medals = {1: "🥇", 2: "🥈", 3: "🥉"}
            rank_emoji = medals.get(i, "🔹")
            description += f"{rank_emoji} **{name}** : ${user['balance']}\n"
        embed.description = description
        return await ctx.send(embed=embed)

    @commands.command(name='top', aliases=['leaderboard', 'classement_gloire'])
    async def glory_leaderboard(self, ctx):
        """Affiche le classement de Gloire."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        from src.database.other import get_top_glory_users
        server_member_ids = [member.id for member in ctx.guild.members]
        users = get_top_glory_users(5, server_member_ids=server_member_ids, server_id=ctx.guild.id)
        if not users:
            return await ctx.send(t("leaderboard.no_glory", lang))
        
        embed = discord.Embed(title=t("leaderboard.glory_title", lang), color=discord.Color.gold())
        description = ""
        for i, user in enumerate(users, 1):
            member = ctx.guild.get_member(int(user["user_id"]))
            name = member.display_name if member else t("leaderboard.unknown", lang)
            medals = {1: "🥇", 2: "🥈", 3: "🥉"}
            rank_emoji = medals.get(i, "🔹")
            description += f"{rank_emoji} **{name}** : {user['glory']} {t('leaderboard.glory_pts', lang)}\n"
            
        embed.description = description
        return await ctx.send(embed=embed)

    @commands.command(name='top_pets', aliases=['pet_leaderboard', 'classement_pets'])
    async def pet_leaderboard(self, ctx):
        """Affiche le classement ELO des familiers."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        from src.database.other import get_top_pets
        from src.models.Pet import PETS_DB
        pets = get_top_pets(5, server_id=ctx.guild.id)
        if not pets:
            return await ctx.send(t("leaderboard.no_pets", lang))
        
        embed = discord.Embed(title=t("leaderboard.pets_title", lang), color=discord.Color.green())
        description = ""
        for i, pet in enumerate(pets, 1):
            member = ctx.guild.get_member(int(pet["user_id"]))
            owner_name = member.display_name if member else t("leaderboard.unknown", lang)
            medals = {1: "🥇", 2: "🥈", 3: "🥉"}
            rank_emoji = medals.get(i, "🔹")
            
            pet_info = PETS_DB.get(pet["pet_type"], {})
            pet_emoji = pet_info.get("emoji", "🐾")
            pet_name = pet["nickname"]
            if pet_name == pet["pet_type"]:
                pet_name = get_pet_name(pet["pet_type"], lang)
            
            description += f"{rank_emoji} **{pet_name}** {pet_emoji} ({owner_name}) : {pet['elo']} ELO\n"
            
        embed.description = description
        return await ctx.send(embed=embed)

async def setup(bot):
    await bot.add_cog(Leaderboard(bot))
