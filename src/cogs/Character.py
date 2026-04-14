import discord
from discord.ext import commands

from src.database.job import get_job_data
from src.database.settings import get_language
from src.utils.i18n import t


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
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user = target if target else ctx.author

        jobs_config = {
            "miner": {"emoji": "⛏️"},
            "fisher": {"emoji": "🎣"},
            "farmer": {"emoji": "🚜"},
            "gambler": {"emoji": "🎰"},
            "crafter": {"emoji": "🛠️"}
        }
        
        embed = discord.Embed(
            title=t("profile.title", lang, user=user.name),
            color=discord.Color.gold()
        )
        embed.set_thumbnail(url=user.display_avatar.url)
        
        total_level = 0
        for job_key, info in jobs_config.items():
            level, current_xp = get_job_data(user.id, job_key)

            xp_needed = get_xp_requirement(level)
            progress_bar = make_progress_bar(current_xp, xp_needed)
            
            job_name = t(f"jobs.{job_key}", lang)
            level_label = t("profile.level_label", lang, level=level)
            xp_label = t("profile.xp_label", lang, current=current_xp, needed=xp_needed)
            
            embed.add_field(
                name=f"{info['emoji']} {job_name} ({level_label})",
                value=f"`{progress_bar}`\n*{xp_label}*",
                inline=False
            )
            total_level += level
            
        embed.set_footer(text=t("profile.footer", lang, total=total_level))
        await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Character(bot))
