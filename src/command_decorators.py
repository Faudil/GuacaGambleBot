from enum import Enum

import discord
import functools

from datetime import datetime

from src.database.limit import check_game_limit, increment_game_limit


def daily_limit(game_name, max_usage):
    def decorator(func):
        @functools.wraps(func)
        async def wrapper(self, ctx, *args, **kwargs):
            user_id = str(ctx.author.id)
            auth, remaining = check_game_limit(user_id, game_name, max_usage)
            if not auth:
                embed = discord.Embed(title="🛑 Limite atteinte", color=discord.Color.red())
                embed.description = f"Tu as déjà joué **{max_usage} fois** à {game_name} aujourd'hui.\nReviens demain !"
                return await ctx.send(embed=embed)
            increment_game_limit(user_id, game_name)
            return await func(self, ctx, *args, **kwargs)
        return wrapper
    return decorator


class ActivityType(Enum):
    MINING = "⛔ **Fermé !** La mine n'ouvre que l'après-midi (de {}h à {}h). Les mineurs boivent .\n* La porte reste fermée.*",
    FISHING = "⛔ **Ça ne mord pas...** Les poissons dorment.\nReviens pêcher le matin (de {}h à {}h).",
    FARMING = "**Fermée** Les plantes ne poussent pas la nuit.\n Revient plutôt en journée {}h à {}h.",
    CASINO = "⛔ **Fermé !** Le Casino n'ouvre que la nuit (de {}h à {}h).\n*Le videur ne te laisse pas entrer.*",


def opening_hours(activity: ActivityType, start_hour: int, end_hour: int):
    def decorator(func):
        @functools.wraps(func)
        async def wrapper(self, ctx, *args, **kwargs):
            current_hour = datetime.now().hour
            is_open = False
            if start_hour < end_hour:
                if start_hour <= current_hour < end_hour:
                    is_open = True
            else:
                if current_hour >= start_hour or current_hour < end_hour:
                    is_open = True
            if is_open:
                return await func(self, ctx, *args, **kwargs)
            else:
                return await ctx.send(str(activity.value).format(start_hour, end_hour))
        return wrapper
    return decorator
