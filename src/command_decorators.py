import discord
from datetime import datetime
import functools
from src.data_handling import check_game_limit, increment_game_limit


def daily_limit(game_name, max_usage):
    def decorator(func):
        @functools.wraps(func)
        async def wrapper(self, ctx, *args, **kwargs):
            user_id = str(ctx.author.id)

            today_str = datetime.now().strftime("%Y-%m-%d")
            auth, remaining = check_game_limit(user_id, today_str, max_usage)
            if not auth:
                embed = discord.Embed(title="🛑 Limite atteinte", color=discord.Color.red())
                embed.description = f"Tu as déjà joué **{max_usage} fois** à {game_name} aujourd'hui.\nReviens demain !"
                return await ctx.send(embed=embed)
            increment_game_limit(user_id, today_str)
            return await func(self, ctx, *args, **kwargs)

        return wrapper

    return decorator