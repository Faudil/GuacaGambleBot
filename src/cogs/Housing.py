import discord
from discord.ext import commands
import datetime
import math

from src.database.housing import (
    get_user_housing, buy_house, upgrade_house, update_last_collected,
    rename_house, set_house_color, start_construction, complete_construction,
    get_crowns, update_crowns, get_housing_upgrades, get_stored_items, update_stored_items,
    get_user_capacities, is_inventory_full, can_add_pet, add_extra_slots
)
from src.database.balance import get_balance, update_balance
from src.database.item import add_item_to_inventory, has_item, remove_item_from_inventory
from src.database.settings import get_language
from src.utils.i18n import t, get_item_name
from src.housing_data import HOUSES, UPGRADES_TREE, BASE_PRODUCTION

class Housing(commands.GroupCog, name="house"):
    def __init__(self, bot):
        self.bot = bot

    def get_progress_bar(self, percent, size=10):
        filled = max(0, min(size, math.floor(size * (percent / 100))))
        return "▓" * filled + "░" * (size - filled)

    def get_house_details(self, house_type, lang):
        house = HOUSES.get(house_type)
        if not house:
            return None
        return {
            "name": t(f"housing.types.{house_type}", lang),
            "price": house["price"],
            "income": house["income_per_hour"],
            "buffs": house["buffs"],
            "color": house["color"],
            "max_level": house["max_level"]
        }

    @commands.hybrid_command(name="show", description="Show your current house status.")
    async def show(self, ctx, member: discord.Member = None):
        user = member or ctx.author
        lang = get_language(ctx.guild.id if ctx.guild else None)
        
        housing_data = get_user_housing(user.id)
        if not housing_data:
            if user == ctx.author:
                return await ctx.send(t("housing.no_house", lang))
            else:
                return await ctx.send(f"{user.display_name} has no home yet.")

        house_type = housing_data['house_type']
        level = housing_data['level']
        details = self.get_house_details(house_type, lang)
        
        # Personalized Title & Color
        title = housing_data['custom_name'] or t("housing.title", lang, user=user.display_name)
        color = int(housing_data['custom_color'], 16) if housing_data['custom_color'] else details['color']
        
        embed = discord.Embed(
            title=f"🏠 {title}",
            description=f"**{details['name']}** (Lvl {level})",
            color=color
        )
        
        # Construction Status
        if housing_data['under_construction']:
            upg = UPGRADES_TREE.get(housing_data['under_construction'])
            finish_time = datetime.datetime.fromisoformat(housing_data['finish_time'])
            now = datetime.datetime.now()
            
            if now >= finish_time:
                embed.add_field(name="🛠️ Construction", value=f"**{upg['name']}** is finished!\nUse `!house complete` to finalize.", inline=False)
            else:
                total_duration = upg['time_hours'] * 3600
                remaining = (finish_time - now).total_seconds()
                percent = 100 - (remaining / total_duration * 100)
                bar = self.get_progress_bar(percent)
                time_left = str(finish_time - now).split('.')[0]
                embed.add_field(name="🛠️ Under Construction", value=f"**{upg['name']}**\n`{bar}` {percent:.1f}%\nReady in: `{time_left}`", inline=False)

        # Passive Rewards
        last_collected = datetime.datetime.fromisoformat(housing_data['last_collected'])
        elapsed = datetime.datetime.now() - last_collected
        hours = elapsed.total_seconds() / 3600
        
        # Income calculation (Base + Level)
        base_income = details['income'] * (1 + (level - 1) * 0.1)
        # Tree bonus (Merchant)
        upgrades = get_housing_upgrades(user.id)
        if "merchant_office" in upgrades: base_income *= 1.15
        
        pending_income = math.floor(hours * base_income)
        
        # Item production
        production = BASE_PRODUCTION.get(house_type, {})
        prod_mult = 2.0 if "industrial_workshop" in upgrades else 1.0
        pending_items = []
        for item_id, rate in production.items():
            qty = math.floor(hours * rate * prod_mult)
            if qty > 0:
                item_name = get_item_name(item_id, lang)
                pending_items.append(f"• {item_name}: `x{qty}`")

        buffs_text = "\n".join([f"• {b}" for b in details['buffs']])
        embed.add_field(name="Stats", value=t("housing.stats", lang, level=level, buffs=buffs_text))
        
        # Capacities
        _, inv_count, inv_max = is_inventory_full(user.id)
        _, pet_count, pet_max = can_add_pet(user.id)
        
        storage_text = f"🎒 **Inventory:** `{inv_count}/{inv_max}`\n🐾 **Pets:** `{pet_count}/{pet_max}`"
        embed.add_field(name="Storage Capacity", value=storage_text)
        
        income_text = f"💰 **${pending_income}** pending\n"
        if pending_items:
            income_text += "\n".join(pending_items)
        
        embed.add_field(name="Pending Rewards", value=income_text)
        
        embed.set_footer(text="Use !house collect to claim your rewards!")
        await ctx.send(embed=embed)

    @commands.hybrid_group(name="expand", description="Expand your storage capacities.")
    async def expand(self, ctx):
        if ctx.invoked_subcommand is None:
            await ctx.send("Usage: `!house expand items` or `!house expand pets`.")

    @expand.command(name="items", description="Add +10 inventory slots for 5 Crowns.")
    async def expand_items(self, ctx):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_crowns = get_crowns(ctx.author.id)
        if user_crowns < 5:
            return await ctx.send(f"❌ You need `5` 👑 Crowns to expand (Current: {user_crowns}).")
            
        update_crowns(ctx.author.id, -5)
        add_extra_slots(ctx.author.id, inv_slots=10)
        await ctx.send(t("housing.expand_items_success", lang))

    @expand.command(name="pets", description="Add +1 pet slot for 10 Crowns.")
    async def expand_pets(self, ctx):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_crowns = get_crowns(ctx.author.id)
        if user_crowns < 10:
            return await ctx.send(f"❌ You need `10` 👑 Crowns to expand (Current: {user_crowns}).")
            
        update_crowns(ctx.author.id, -10)
        add_extra_slots(ctx.author.id, pet_slots=1)
        await ctx.send(t("housing.expand_pets_success", lang))

    @commands.hybrid_command(name="tree", description="View the house upgrade tree.")
    async def tree(self, ctx):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_upgrades = get_housing_upgrades(ctx.author.id)
        
        embed = discord.Embed(
            title="🌳 House Upgrade Tree",
            description="Choose a path to specialize your home.",
            color=discord.Color.dark_green()
        )
        
        for upg_id, upg in UPGRADES_TREE.items():
            status = "✅ Purchased" if upg_id in user_upgrades else "🔒 Locked"
            if not upg['requires'] or upg['requires'] in user_upgrades:
                if upg_id not in user_upgrades:
                    status = f"💰 Cost: ${upg['cost_money']} + Resources"
                
                items_req = ", ".join([f"{qty}x {get_item_name(i, lang)}" for i, qty in upg['cost_items'].items()])
                embed.add_field(
                    name=f"{upg['name']} ({upg['branch'].capitalize()})",
                    value=f"Status: **{status}**\nReqs: {items_req}\nTime: {upg['time_hours']}h\n*Effect: {upg['bonus_desc']}*",
                    inline=False
                )
        
        await ctx.send(embed=embed)

    @commands.hybrid_command(name="upgrade", description="Start an upgrade.")
    async def upgrade_cmd(self, ctx, upgrade_id: str = None):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        housing_data = get_user_housing(ctx.author.id)
        if not housing_data:
            return await ctx.send(t("housing.no_house", lang))
            
        if housing_data['under_construction']:
            return await ctx.send("❌ You already have an active construction project!")

        # If no ID, perform standard level upgrade
        if not upgrade_id:
            # Linear level upgrade logic (money only for now, as before)
            level = housing_data['level']
            details = HOUSES[housing_data['house_type']]
            if level >= details['max_level']:
                return await ctx.send(t("housing.max_level", lang))
            
            cost = math.floor(details['price'] * 0.5 * (1.5 ** (level - 1)))
            if get_balance(ctx.author.id) < cost:
                return await ctx.send(f"❌ You need **${cost}** to upgrade.")
            
            update_balance(ctx.author.id, -cost)
            upgrade_house(ctx.author.id)
            return await ctx.send(t("housing.upgrade_success", lang, level=level+1))

        # Branching Tree Upgrade
        if upgrade_id not in UPGRADES_TREE:
            return await ctx.send("Invalid upgrade ID.")
            
        upg = UPGRADES_TREE[upgrade_id]
        user_upgrades = get_housing_upgrades(ctx.author.id)
        
        if upgrade_id in user_upgrades:
            return await ctx.send("❌ You already have this upgrade!")
            
        if upg['requires'] and upg['requires'] not in user_upgrades:
            return await ctx.send(f"❌ You need to unlock **{UPGRADES_TREE[upg['requires']]['name']}** first!")

        # Check costs
        if get_balance(ctx.author.id) < upg['cost_money']:
            return await ctx.send(f"❌ You need **${upg['cost_money']}**.")
            
        for item_id, qty in upg['cost_items'].items():
            if not has_item(ctx.author.id, item_id, qty):
                return await ctx.send(f"❌ Missing resources: **{qty}x {get_item_name(item_id, lang)}**")

        # Deduct costs
        update_balance(ctx.author.id, -upg['cost_money'])
        for item_id, qty in upg['cost_items'].items():
            remove_item_from_inventory(ctx.author.id, item_id, qty)
            
        # Start timer
        start_construction(ctx.author.id, upgrade_id, upg['time_hours'])
        await ctx.send(f"🏗️ **Construction Started!** {upg['name']} will be ready in {upg['time_hours']} hours.")

    @commands.hybrid_command(name="complete", description="Finalize a finished construction.")
    async def complete(self, ctx):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        housing_data = get_user_housing(ctx.author.id)
        if not housing_data or not housing_data['under_construction']:
            return await ctx.send("❌ No active construction.")
            
        finish_time = datetime.datetime.fromisoformat(housing_data['finish_time'])
        if datetime.datetime.now() < finish_time:
            time_left = str(finish_time - datetime.datetime.now()).split('.')[0]
            return await ctx.send(f"⏳ Construction is not finished! Time remaining: `{time_left}`")
            
        complete_construction(ctx.author.id)
        await ctx.send("🎉 **Upgrade Complete!** Your home has been successfully improved.")

    @commands.hybrid_command(name="speedup", description="Complete construction using Crowns.")
    async def speedup(self, ctx):
        housing_data = get_user_housing(ctx.author.id)
        if not housing_data or not housing_data['under_construction']:
            return await ctx.send("❌ No active construction to speed up.")
            
        finish_time = datetime.datetime.fromisoformat(housing_data['finish_time'])
        now = datetime.datetime.now()
        if now >= finish_time:
            return await ctx.send("✅ It's already finished! Use `!house complete`.")
            
        remaining_hours = math.ceil((finish_time - now).total_seconds() / 3600)
        crown_cost = max(1, remaining_hours)
        
        user_crowns = get_crowns(ctx.author.id)
        if user_crowns < crown_cost:
            return await ctx.send(f"❌ You need `{crown_cost}` 👑 Crowns to speed this up (Current: {user_crowns}).")
            
        update_crowns(ctx.author.id, -crown_cost)
        complete_construction(ctx.author.id)
        await ctx.send(f"⚡ **Construction Rushed!** Spent `{crown_cost}` 👑 Crowns. Your upgrade is ready!")

    @commands.hybrid_command(name="collect", description="Collect pending rent and resources.")
    async def collect(self, ctx):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        housing_data = get_user_housing(ctx.author.id)
        if not housing_data:
            return await ctx.send(t("housing.no_house", lang))
            
        house_type = housing_data['house_type']
        production = BASE_PRODUCTION.get(house_type, {})
        
        # Check if inventory is full BEFORE collecting items
        if production:
            from src.database.housing import is_inventory_full
            full, current, limit = is_inventory_full(ctx.author.id)
            if full:
                return await ctx.send(t("housing.inv_full_warning", lang, current=current, limit=limit))
        
        level = housing_data['level']
        details = HOUSES[house_type]
        upgrades = get_housing_upgrades(ctx.author.id)
        
        last_collected = datetime.datetime.fromisoformat(housing_data['last_collected'])
        elapsed = datetime.datetime.now() - last_collected
        hours = elapsed.total_seconds() / 3600
        
        # Calculate Income
        base_income = details['income_per_hour'] * (1 + (level - 1) * 0.1)
        if "merchant_office" in upgrades: base_income *= 1.15
        income = math.floor(hours * base_income)
        
        # Calculate Items
        production = BASE_PRODUCTION.get(house_type, {})
        prod_mult = 2.0 if "industrial_workshop" in upgrades else 1.0
        collected_items = []
        for item_id, rate in production.items():
            qty = math.floor(hours * rate * prod_mult)
            if qty > 0:
                add_item_to_inventory(ctx.author.id, item_id, qty)
                collected_items.append(f"{qty}x {get_item_name(item_id, lang)}")
        
        # Rare Resources (Industrial Drill)
        if "industrial_drill" in upgrades and hours >= 1:
            # 5% chance per hour of getting a rare ore
            import random
            if random.random() < (0.05 * hours):
                rare_ores = ["diamant brut", "émeraude", "platine"]
                rare_ore = random.choice(rare_ores)
                add_item_to_inventory(ctx.author.id, rare_ore, 1)
                collected_items.append(f"⭐ 1x {get_item_name(rare_ore, lang)}")

        if income <= 0 and not collected_items:
            return await ctx.send(t("housing.nothing_to_collect", lang))
        
        update_balance(ctx.author.id, income)
        update_last_collected(ctx.author.id)
        
        msg = f"💰 **Collected!** +${income}"
        if collected_items:
            msg += f"\n📦 **Resources:** {', '.join(collected_items)}"
        
        await ctx.send(msg)

    @commands.hybrid_command(name="rename", description="Give your home a custom name.")
    async def rename(self, ctx, *, name: str):
        if len(name) > 32:
            return await ctx.send("❌ Name too long (max 32 chars).")
        rename_house(ctx.author.id, name)
        await ctx.send(f"✅ Your home is now known as: **{name}**")

    @commands.hybrid_command(name="color", description="Set a custom theme color (Hex code).")
    async def color(self, ctx, hex_code: str):
        if not hex_code.startswith("#") or len(hex_code) != 7:
            return await ctx.send("❌ Invalid format. Use `#RRGGBB`.")
        
        # Remove # and convert to hex
        clean_hex = hex_code.lstrip('#')
        try:
            int(clean_hex, 16)
        except ValueError:
            return await ctx.send("❌ Invalid hex code.")
            
        set_house_color(ctx.author.id, clean_hex)
        await ctx.send(f"🎨 Theme color updated to **{hex_code}**!")

async def setup(bot):
    await bot.add_cog(Housing(bot))
