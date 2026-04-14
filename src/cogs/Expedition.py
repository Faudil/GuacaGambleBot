import discord
from discord.ext import commands
import json
from datetime import datetime, timedelta
import math

from src.database.pets import get_active_pet, update_pet, get_pet_by_id
from src.database.expedition import (
    start_expedition, get_active_expedition, claim_expedition
)
from src.database.item import add_item_to_inventory
from src.database.settings import get_language
from src.utils.i18n import t, get_item_name
from src.utils.ExpeditionManager import generate_expedition

class Expedition(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='expedition', aliases=['exped'])
    async def expedition_base(self, ctx, sub: str = None, duration: str = None):
        """Système d'expéditions pour familiers."""
        lang = get_language(ctx.guild.id if ctx.guild else None)

        if duration in ["short", "rapide"]:
            duration = 2
        elif duration in ["medium", "moyen"]:
            duration = 6
        elif duration in ["long"]:
            duration = 24
        else:
            return await ctx.send(t("expedition.invalid_duration", lang))

        if sub == "start":
            await self.start_cmd(ctx, duration, lang)
        elif sub == "status":
            await self.status_cmd(ctx, lang)
        elif sub == "claim":
            await self.claim_cmd(ctx, lang)
        else:
            # Help or prompt
            embed = discord.Embed(title="🚀 " + t("expedition.help_title", lang), color=discord.Color.blue())
            embed.description = (
                t("expedition.help_desc", lang) + "\n\n" +
                t("expedition.help_benefits", lang) + "\n" +
                t("expedition.help_benefits_list", lang) + "\n\n" +
                t("expedition.help_commands", lang) + "\n" +
                t("expedition.help_commands_list", lang)
            )
            return await ctx.send(embed=embed)

    async def start_cmd(self, ctx, duration: int, lang: str):
        if duration not in [1, 4, 8, 24]:
            return await ctx.send(t("expedition.invalid_duration", lang))
            
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send(t("expedition.no_pet", lang))
            
        active = get_active_expedition(ctx.author.id)
        if active:
            return await ctx.send(t("expedition.already_active", lang))
            
        data = generate_expedition(pet, duration, lang)
        
        start_expedition(
            ctx.author.id, pet.id, duration, 
            data["xp"], data["items"], data["log"]
        )
        
        await ctx.send(t("expedition.start_success", lang, pet=pet.nickname, duration=duration))

    async def status_cmd(self, ctx, lang):
        exp = get_active_expedition(ctx.author.id)
        if not exp:
            return await ctx.send(t("expedition.no_active", lang))
            
        pet = get_pet_by_id(exp['pet_id'])
        
        start_dt = datetime.fromisoformat(exp['start_time'])
        end_dt = datetime.fromisoformat(exp['end_time'])
        now = datetime.now()
        
        total_duration = (end_dt - start_dt).total_seconds()
        elapsed = (now - start_dt).total_seconds()
        
        if elapsed < 0: elapsed = 0
        
        progress = min(100, max(0, int((elapsed / total_duration) * 100)))
        
        embed = discord.Embed(
            title=t("expedition.status_title", lang, pet=pet.nickname),
            color=discord.Color.green() if progress == 100 else discord.Color.orange()
        )
        
        remaining_str = ""
        if progress < 100:
            remaining = end_dt - now
            hours, remainder = divmod(int(remaining.total_seconds()), 3600)
            minutes, seconds = divmod(remainder, 60)
            remaining_str = t("expedition.time_format", lang, hours=hours, minutes=minutes)

        embed.description = t("expedition.status_desc", lang, progress=progress, end_time=end_dt.strftime("%H:%M"))
        
        log = json.loads(exp['log'])
        elapsed_mins = elapsed / 60
        visible_events = [e for e in log if e['time'] <= elapsed_mins]
        
        if not visible_events:
            log_text = t("expedition.no_events", lang)
        else:
            log_text = ""
            for e in visible_events[-8:]:
                time_str = f"[{e['time']}m]" # Time in minutes relative to start, usually not localized
                log_text += f"{time_str} {e['text']}\n"
                
        embed.add_field(name=t("expedition.log_header", lang), value=log_text, inline=False)
        
        if progress == 100:
            embed.add_field(name="✨ " + t("expedition.claim_ready", lang), value="\u200b")
            
        await ctx.send(embed=embed)

    async def claim_cmd(self, ctx, lang):
        exp = get_active_expedition(ctx.author.id)
        if not exp:
            return await ctx.send(t("expedition.no_active", lang))
            
        end_dt = datetime.fromisoformat(exp['end_time'])
        now = datetime.now()
        if now < end_dt:
            # Not finished yet
            remaining = end_dt - now
            hours, remainder = divmod(int(remaining.total_seconds()), 3600)
            minutes, seconds = divmod(remainder, 60)
            rem_str = t("expedition.time_format", lang, hours=hours, minutes=minutes)
            return await ctx.send(t("expedition.not_finished", lang, remaining=rem_str))
            
        pet = get_pet_by_id(exp['pet_id'])

        # Give rewards
        reward_xp = exp['reward_xp']
        items = json.loads(exp['reward_items'])
        
        leveled_up = pet.add_xp(reward_xp)
        update_pet(pet)
        
        looted_str = ""
        if items:
            # Group items for cleaner display
            from collections import Counter
            counts = Counter(items)
            for item_name, count in counts.items():
                add_item_to_inventory(ctx.author.id, item_name.lower(), count)
                looted_str += f"- {count}x {get_item_name(item_name, lang)}\n"
        else:
            looted_str = t("expedition.no_items", lang)
            
        claim_expedition(exp['id'])
        
        embed = discord.Embed(title=t("expedition.claim_title", lang), color=discord.Color.gold())
        embed.description = t("expedition.claim_success", lang, pet=pet.nickname, xp=reward_xp, items="\n" + looted_str)
        
        if leveled_up:
            embed.description += "\n\n" + t("pets.play.level_up", lang, name=pet.nickname, level=pet.level)
            
        await ctx.send(embed=embed)

async def setup(bot):
    await bot.add_cog(Expedition(bot))
