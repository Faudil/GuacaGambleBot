import discord
from discord.ext import commands

from src.database.item import has_item, remove_item_from_inventory, add_item_to_inventory
from src.database.job import get_job_data, add_job_xp
from src.database.settings import get_language
from src.database.housing import get_user_housing
from src.housing_data import HOUSES
from src.items.Recipe import RECIPES
from src.utils.i18n import t, get_item_name
import math


class Crafting(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    def resolve_item_key(self, display_name: str, lang: str) -> str:
        """Tentative de résolution d'un nom traduit vers la clé technique de RECIPES."""
        search = display_name.lower().strip()
        # Direct match with key
        if search in RECIPES:
            return search
        
        # Match with localized name
        for key in RECIPES.keys():
            if get_item_name(key, lang).lower() == search:
                return key
                
        return search # Return as is if not found, will fail later anyway

    @commands.command(name='recipes', aliases=['recettes', 'crafting'])
    async def recipes(self, ctx):
        """Affiche la liste des recettes de craft."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        level, _ = get_job_data(ctx.author.id, "crafter")

        unlocked = []
        locked = []
        for name, recipe in RECIPES.items():
            ing_strs = []
            for ing_name, qty in recipe.ingredients.items():
                ing_strs.append(f"{qty}x {get_item_name(ing_name, lang)}")
            ing_str = ", ".join(ing_strs)
            
            res_name = get_item_name(recipe.result_item, lang)
            if recipe.level_required <= level:
                unlocked.append(t("crafting.unlock_line", lang, item=res_name, ingredients=ing_str))
            else:
                locked.append(t("crafting.lock_line", lang, item=res_name, level=recipe.level_required, ingredients=ing_str))
                
        description = t("crafting.desc_intro", lang)
        
        description += t("crafting.unlocked_title", lang)
        if unlocked:
            description += "\n".join(unlocked) + "\n\n"
        else:
            description += t("crafting.no_unlocked", lang)
            
        if locked:
            description += t("crafting.locked_title", lang) + "\n".join(locked)
            
        embed = discord.Embed(
            title=t("crafting.title", lang, level=level),
            description=description,
            color=discord.Color.orange()
        )
            
        await ctx.send(embed=embed)

    @commands.command(name='craft', aliases=['fabriquer'])
    async def craft(self, ctx, *, args: str = None):
        """Fabrique un objet. Ex: !craft 2 café"""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if not args:
            return await ctx.send(t("crafting.no_args", lang))
            
        args = args.strip()
        amount = 1
        parts = args.split()
        
        if len(parts) > 1 and parts[0].isdigit():
            amount = int(parts[0])
            item_query = ' '.join(parts[1:]).lower()
        elif len(parts) > 1 and parts[-1].isdigit():
            amount = int(parts[-1])
            item_query = ' '.join(parts[:-1]).lower()
        else:
            item_query = args.lower()
            
        if amount <= 0:
            return await ctx.send(t("crafting.invalid_qty", lang))
            
        item_name = self.resolve_item_key(item_query, lang)
            
        if item_name not in RECIPES:
            return await ctx.send(t("crafting.no_recipe", lang, item=item_query))
            
        from src.database.housing import is_inventory_full
        full, current, limit = is_inventory_full(ctx.author.id)
        if current + amount > limit:
            return await ctx.send(t("housing.inv_full_warning", lang, current=current, limit=limit))
            
        recipe = RECIPES[item_name]
        level, _ = get_job_data(ctx.author.id, "crafter")
        
        if level < recipe.level_required:
            return await ctx.send(t("crafting.no_level", lang, level=recipe.level_required))
            
        # Check ingredients and apply housing discount
        housing_data = get_user_housing(ctx.author.id)
        discount = 0
        if housing_data:
            house_type = housing_data['house_type']
            if house_type in HOUSES:
                discount = HOUSES[house_type].get('crafting_discount', 0)
            
            # Tree bonuses
            from src.database.housing import get_housing_upgrades
            upgrades = get_housing_upgrades(ctx.author.id)
            if "mystic_altar" in upgrades: discount += 0.05
            if "mystic_laboratory" in upgrades: discount += 0.15

        missing = []
        ingredients_to_consume = {}
        for ing_name, qty in recipe.ingredients.items():
            req_qty = qty * amount
            if discount > 0:
                req_qty = max(1, math.floor(req_qty * (1 - discount)))
            
            ingredients_to_consume[ing_name] = req_qty
            if not has_item(ctx.author.id, ing_name, req_qty):
                missing.append(f"{req_qty}x {get_item_name(ing_name, lang)}")
                
        if missing:
            return await ctx.send(t("crafting.no_ingredients", lang, missing=', '.join(missing)))
            
        # Consume ingredients
        for ing_name, req_qty in ingredients_to_consume.items():
            remove_item_from_inventory(ctx.author.id, ing_name, req_qty)
            
        # Give item
        add_item_to_inventory(ctx.author.id, recipe.result_item, amount)
        
        # Give XP
        leveled_up, new_lvl = add_job_xp(ctx.author.id, "crafter", recipe.xp_reward * amount)
        
        res_display = get_item_name(recipe.result_item, lang)
        msg = t("crafting.success_msg", lang, amount=amount, item=res_display, xp=recipe.xp_reward * amount)
        if leveled_up:
            msg += t("crafting.level_up", lang, level=new_lvl)
            
        await ctx.send(msg)


async def setup(bot):
    await bot.add_cog(Crafting(bot))