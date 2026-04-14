import discord
from discord.ext import commands
import math
from src.utils.i18n import t
from src.database.community import get_server_projects, get_project_level, get_project_contributions, add_project_contribution, set_project_level, reset_project_contributions, get_user_community_stats, add_user_community_stats
from src.utils.community_data import BUILDINGS, get_building
from src.database.balance import get_balance, update_balance
from src.database.item import has_item, remove_item_from_inventory
from src.database.settings import get_language

class CommunityProjects(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @discord.slash_command(name="community", description="Manage community buildings and check progress.")
    async def community_base(self, ctx):
        pass

    @community_base.command(name="list", description="List all community buildings and their active bonuses.")
    async def list_buildings(self, ctx):
        server_id = ctx.guild.id
        from src.database.settings import get_language
        lang = get_language(server_id)

        completed_projects = get_server_projects(server_id)
        
        embed = discord.Embed(title=t("community.list_title", lang), color=discord.Color.brand_green())
        
        for key, building in BUILDINGS.items():
            level = completed_projects.get(key, 0)
            target_level = level + 1 if level < building.max_level else building.max_level
            
            b_name = t(f"community.building_{key}_name", lang)
            b_desc = t(f"community.building_{key}_desc", lang)
            
            b_info = f"**{b_name}** (Lvl {level}/{building.max_level})\n"
            b_info += f"_{b_desc}_\n"
            
            bonuses = building.get_bonuses(level)
            if bonuses:
                for b_k, b_v in bonuses.items():
                    bonus_text = t(f"community.bonus_{b_k}", lang, val=b_v)
                    b_info += f"✅ {bonus_text}\n"
            else:
                b_info += f"❌ " + t("community.no_bonus_yet", lang) + "\n"
                
            embed.add_field(name="\u200b", value=b_info, inline=False)
            
        await ctx.respond(embed=embed)

    def generate_progress_bar(self, current, max_val, length=10):
        if max_val == 0:
            return "🟦" * length
        perc = current / max_val
        filled = int(length * perc)
        filled = min(length, filled)
        empty = length - filled
        return "🟦" * filled + "⬜" * empty

    @community_base.command(name="inspect", description="Inspect a specific community building to see cost and progress.")
    async def inspect_building(self, ctx, building_name: str):
        server_id = ctx.guild.id
        lang = get_language(server_id)

        building_name = building_name.lower()
        if building_name not in BUILDINGS:
            # Try to lookup by translated name
            found = None
            for k in BUILDINGS.keys():
                if t(f"community.building_{k}_name", lang).lower() == building_name:
                    found = k
                    break
            if found:
                building_name = found
            else:
                await ctx.respond(t("community.not_found", lang, name=building_name))
                return

        building = BUILDINGS[building_name]
        level = get_project_level(server_id, building_name)
        
        # Determine actual level based on DB (1 is default, but completed check needs to happen)
        actual_level = get_server_projects(server_id).get(building_name, 0)
        target_level = actual_level + 1
        
        if actual_level >= building.max_level:
            await ctx.respond(t("community.max_level", lang, name=t(f"community.building_{building_name}_name", lang)))
            return
            
        costs = building.get_cost(target_level)
        contributions = get_project_contributions(server_id, building_name)
        
        b_name = t(f"community.building_{building_name}_name", lang)
        embed = discord.Embed(title=t("community.inspect_title", lang, name=b_name, level=target_level), color=discord.Color.blue())
        
        for res, required in costs.items():
            current = contributions.get(res, 0)
            res_name = t("community.res_money", lang) if res == "money" else res
            
            bar = self.generate_progress_bar(current, required)
            perc = math.floor((current/required)*100) if required > 0 else 100
            perc = min(100, perc)
            
            embed.add_field(name=f"{res_name} ({current}/{required})", value=f"{bar} {perc}%", inline=False)
            
        await ctx.respond(embed=embed)

    @community_base.command(name="invest", description="Invest resources into a community building.")
    async def invest_building(self, ctx, building_name: str, res_key: str, amount: int):
        server_id = ctx.guild.id
        user_id = ctx.author.id
        lang = get_language(server_id)

        if amount <= 0:
            return await ctx.respond(t("error.invalid_amount", lang))

        building_name = building_name.lower()
        res_key = res_key.lower()
        if building_name not in BUILDINGS:
            # Fallback translate check
            found = None
            for k in BUILDINGS.keys():
                if t(f"community.building_{k}_name", lang).lower() == building_name:
                    found = k
                    break
            if found:
                building_name = found
            else:
                return await ctx.respond(t("community.not_found", lang, name=building_name))

        building = BUILDINGS[building_name]
        actual_level = get_server_projects(server_id).get(building_name, 0)
        target_level = actual_level + 1
        
        if actual_level >= building.max_level:
            return await ctx.respond(t("community.max_level", lang, name=t(f"community.building_{building_name}_name", lang)))

        costs = building.get_cost(target_level)
        
        if res_key not in costs:
            return await ctx.respond(t("community.res_not_needed", lang, res=res_key))

        contributions = get_project_contributions(server_id, building_name)
        current = contributions.get(res_key, 0)
        required = costs[res_key]

        if current >= required:
            return await ctx.respond(t("community.res_already_full", lang, res=res_key))
            
        invest_amount = min(amount, required - current)
        
        if res_key == "money":
            bal = get_balance(user_id)
            if bal < invest_amount:
                return await ctx.respond(t("economy.not_enough_money", lang))
            update_balance(user_id, -invest_amount)
            add_user_community_stats(user_id, server_id, money=invest_amount)
        else:
            inv_qty = has_item(user_id, res_key, amount)
            if inv_qty < invest_amount:
                return await ctx.respond(t("inventory.not_enough_item", lang, item=res_key))
            remove_item_from_inventory(user_id, res_key, invest_amount)
            add_user_community_stats(user_id, server_id, items=invest_amount)
            
        add_project_contribution(server_id, building_name, res_key, invest_amount)

        # Check if building leveled up
        contributions = get_project_contributions(server_id, building_name)
        level_up = True
        for c_res, c_req in costs.items():
            c_cur = contributions.get(c_res, 0)
            if c_cur < c_req:
                level_up = False
                break
                
        if level_up:
            set_project_level(server_id, building_name, target_level)
            reset_project_contributions(server_id, building_name)
            b_name = t(f"community.building_{building_name}_name", lang)
            embed = discord.Embed(title=t("community.level_up_title", lang), description=t("community.level_up_desc", lang, name=b_name, level=target_level), color=discord.Color.gold())
            await ctx.respond(embed=embed)
        else:
            await ctx.respond(t("community.invest_success", lang, amount=invest_amount, res=res_key, building=t(f"community.building_{building_name}_name", lang)))
        return None

    @community_base.command(name="stats", description="View your total investments in this server's community.")
    async def community_stats(self, ctx):
        server_id = ctx.guild.id
        lang = get_language(server_id)
        
        stats = get_user_community_stats(ctx.author.id, server_id)
        embed = discord.Embed(title=t("community.stats_title", lang, user=ctx.author.display_name), color=discord.Color.purple())
        embed.add_field(name=t("community.res_money", lang), value=f"{stats['money']:,}")
        embed.add_field(name=t("community.res_items", lang), value=f"{stats['items']:,}")
        await ctx.respond(embed=embed)

def setup(bot):
    bot.add_cog(CommunityProjects(bot))
