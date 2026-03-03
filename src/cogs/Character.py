import discord
from discord.ext import commands

from src.database.job import get_job_data


def make_progress_bar(xp, xp_needed, length=10):
    if xp_needed == 0: xp_needed = 1

    percent = min(1.0, xp / xp_needed)
    filled_length = int(length * percent)
    empty_length = length - filled_length

    bar = "█" * filled_length + "░" * empty_length
    return f"[{bar}] {int(percent * 100)}%"


def get_xp_requirement(level):
    return level * 100


class Character(commands.Cog):
    def __init__(self, bot:commands.Bot):
        self.bot = bot

    @commands.command(name='level', aliases=['levels', 'jobstats', 'profil', 'skills', 'lvl'])
    async def level(self, ctx, target: discord.Member = None):
        """Affiche le niveau et les compétences d'un joueur."""
        user = target if target else ctx.author

        jobs_config = {
            "miner": {"emoji": "⛏️", "name": "Mineur", "color": "⬜"},
            "fisher": {"emoji": "🎣", "name": "Pêcheur", "color": "🟦"},
            "farmer": {"emoji": "🚜", "name": "Fermier", "color": "🟨"},
            "gambler": {"emoji": "🎰", "name": "Gambleur", "color": "🟨"},
            # "crafter": {"emoji": "🛠️", "name": "Artisan", "color": "🟧"}
        }
        embed = discord.Embed(
            title=f"📊 Compétences de {user.name}",
            color=discord.Color.gold()
        )
        embed.set_thumbnail(url=user.avatar.url if user.avatar else None)
        total_level = 0
        for job_key, info in jobs_config.items():
            level, current_xp = get_job_data(user.id, job_key)

            xp_needed = get_xp_requirement(level)
            progress_bar = make_progress_bar(current_xp, xp_needed)
            embed.add_field(
                name=f"{info['emoji']} {info['name']} (Niv. {level})",
                value=f"`{progress_bar}`\n*XP : {current_xp} / {xp_needed}*",
                inline=False
            )
            total_level += level
        embed.set_footer(text=f"Niveau Global : {total_level} | Continue de bosser !")
        await ctx.send(embed=embed)

async def setup(bot):
    await bot.add_cog(Character(bot))
