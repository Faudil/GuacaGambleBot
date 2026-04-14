import discord
from discord.ext import commands
import random
import asyncio

from src.database.achievement import check_and_unlock_achievements, format_achievements_unlocks
from src.database.balance import update_balance, get_balance
from src.database.item import has_item, remove_item_from_inventory, get_item_name_by_id
from src.database.pets import (
    get_all_pets, set_active_pet, get_active_pet, insert_new_pet,
    update_pet, get_pet_by_id, transfer_pet,
    update_pet_elo, delete_pet
)
from src.globals import ITEMS_REGISTRY
from src.items.ForgetPotion import ForgetPotion
from src.items.Item import ItemRarity, Item
from src.items.MysteryEgg import MysteryEgg
from src.models.Pet import PETS_DB, Pet, PetBonus
from src.utils.embed_utils import generate_hp_bar
from src.utils.battle import simulate_battle
from src.utils.i18n import t, get_pet_name, get_rarity_name
from src.database.settings import get_language


class BattleAcceptView(discord.ui.View):
    def __init__(self, challenger, opponent, bet, lang):
        super().__init__(timeout=60.0)
        self.challenger = challenger
        self.opponent = opponent
        self.bet = bet
        self.lang = lang
        self.accepted = None

        self.accept.label = t("pets.battle.accept_label", lang)
        self.decline.label = t("pets.battle.decline_label", lang)

    @discord.ui.button(style=discord.ButtonStyle.success, emoji="⚔️")
    async def accept(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.opponent.id:
            return await interaction.response.send_message(t("pets.battle.wrong_opponent", self.lang), ephemeral=True)
        self.accepted = True
        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(content=t("pets.battle.accepted_msg", self.lang), view=self)
        self.stop()

    @discord.ui.button(style=discord.ButtonStyle.danger, emoji="🏃")
    async def decline(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.opponent.id:
            return await interaction.response.send_message(t("pets.battle.wrong_opponent", self.lang), ephemeral=True)
        self.accepted = False
        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(content=t("pets.battle.refused_msg", self.lang, name=self.opponent.display_name), view=self)
        self.stop()

    async def on_timeout(self):
        self.accepted = False
        for child in self.children:
            child.disabled = True
        try:
            await self.message.edit(content=t("pets.battle.timeout_msg", self.lang), view=self)
        except:
            pass


class PetSellAcceptView(discord.ui.View):
    def __init__(self, seller, buyer, pet, price, lang):
        super().__init__(timeout=60.0)
        self.seller = seller
        self.buyer = buyer
        self.pet = pet
        self.price = price
        self.lang = lang
        self.accepted = None

        self.accept.label = t("pets.sell.buy_label", lang)
        self.decline.label = t("pets.sell.refuse_label", lang)

    @discord.ui.button(style=discord.ButtonStyle.success, emoji="💰")
    async def accept(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.buyer.id:
            return await interaction.response.send_message(t("pets.sell.wrong_buyer", self.lang), ephemeral=True)
        if get_balance(self.buyer.id) < self.price:
             return await interaction.response.send_message(t("pets.sell.buyer_no_money_ephemeral", self.lang), ephemeral=True)
        self.accepted = True
        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(content=t("pets.sell.accepted_msg", self.lang), view=self)
        self.stop()

    @discord.ui.button(style=discord.ButtonStyle.danger, emoji="✖️")
    async def decline(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.buyer.id:
            return await interaction.response.send_message(t("pets.sell.wrong_buyer", self.lang), ephemeral=True)
        self.accepted = False
        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(content=t("pets.sell.refused_msg", self.lang, name=self.buyer.display_name), view=self)
        self.stop()

    async def on_timeout(self):
        self.accepted = False
        for child in self.children:
            child.disabled = True
        try:
            await self.message.edit(content=t("pets.sell.timeout_msg", self.lang), view=self)
        except:
            pass


class Pets(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    def roll_gacha_pet(self, target_rarity=None):
        if not target_rarity:
            roll = random.random()
            if roll < 0.05: target_rarity = ItemRarity.legendary
            elif roll < 0.15: target_rarity = ItemRarity.epic
            elif roll < 0.40: target_rarity = ItemRarity.rare
            else: target_rarity = ItemRarity.common
        possible_pets = [name for name, data in PETS_DB.items() if data["rarity"] == target_rarity]
        return random.choice(possible_pets)

    @commands.command(name='hatch', aliases=['eclore'])
    async def hatch(self, ctx):
        """Faire éclore un Œuf Mystère."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = ctx.author.id
        
        from src.database.housing import can_add_pet
        can, current, limit = can_add_pet(user_id)
        if not can:
            return await ctx.send(t("housing.pet_full_warning", lang, current=current, limit=limit))
            
        me_name = MysteryEgg().name
        if not has_item(user_id, me_name, 1):
            return await ctx.send(t("pets.hatch.no_egg", lang))

        remove_item_from_inventory(user_id, me_name, 1)
        embed = discord.Embed(title=t("pets.hatch.hatching_title", lang), color=discord.Color.light_grey())
        embed.description = t("pets.hatch.step1", lang)
        msg = await ctx.send(embed=embed)

        await asyncio.sleep(2)
        embed.description = t("pets.hatch.step2", lang)
        await msg.edit(embed=embed)

        await asyncio.sleep(2)
        pet_name = self.roll_gacha_pet()
        pet_data = PETS_DB[pet_name]
        insert_new_pet(Pet.create_new(user_id, pet_name, pet_name))

        color = discord.Color.blue()
        if pet_data["rarity"] == ItemRarity.legendary: color = discord.Color.gold()
        elif pet_data["rarity"] == ItemRarity.epic: color = discord.Color.purple()

        loc_pet_name = get_pet_name(pet_name, lang)
        loc_rarity = get_rarity_name(pet_data['rarity'].name, lang)
        loc_bonus = t(f"bonuses.{pet_data['bonus'].name.lower()}", lang)

        embed.title = t("pets.hatch.success_title", lang, emoji=pet_data['emoji'])
        embed.color = color
        embed.description = t("pets.hatch.success_desc", lang, name=loc_pet_name, rarity=loc_rarity, bonus=loc_bonus)
        await msg.edit(embed=embed)

    @commands.command(name='pets', aliases=['familiers', 'zoo'])
    async def my_pets(self, ctx, user: discord.Member = None):
        """Voir ta collection de familiers."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user = user or ctx.author
        server_id = ctx.guild.id if ctx.guild else None
        
        from src.database.housing import can_add_pet
        _, current, limit = can_add_pet(user.id)
        
        target_pets = get_all_pets(user.id, server_id)
        if not target_pets:
            return await ctx.send(t("pets.list.no_pets", lang) + f"\n🐾 **Slots: `{current}/{limit}`**")
        embed = discord.Embed(title=t("pets.list.title", lang, name=user.name) + f" ({current}/{limit})", color=discord.Color.green())
        desc = ""
        for pet in target_pets:
            status_text = t("pets.list.active", lang) if pet.is_active else t("pets.list.inactive", lang)
            desc += f"{pet.emoji} **{pet.nickname}** - {status_text} ID: `{pet.id}`\n"
        embed.description = desc
        embed.set_footer(text=t("pets.list.footer", lang))
        return await ctx.send(embed=embed)

    @commands.command(name='equip')
    async def equip(self, ctx, pet_id: int):
        """Équiper un familier actif."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        success = set_active_pet(ctx.author.id, pet_id)
        if success: await ctx.send(t("pets.equip.success", lang))
        else: await ctx.send(t("pets.equip.fail", lang))

    @commands.command(name='pet_rename', aliases=["petrename", "pname"])
    async def rename_pet(self, ctx, *, new_name: str):
        """Renommer ton familier actif."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if len(new_name) > 20: return await ctx.send(t("pets.rename.too_long", lang))
        server_id = ctx.guild.id if ctx.guild else None
        pet = get_active_pet(ctx.author.id, server_id)
        if not pet: return await ctx.send(t("pets.play.no_pet", lang))
        pet.nickname = new_name
        update_pet(pet)
        await ctx.send(t("pets.rename.success", lang, name=new_name))

    @commands.command(name='sell_pet', aliases=['vendre_familier', "pet_sell", "petsell"])
    async def sell_pet(self, ctx, buyer: discord.Member, pet_id: int, price: int):
        """Vendre un familier à un autre utilisateur."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if ctx.author.id == buyer.id: return await ctx.send(t("pets.sell.self_sell", lang))
        if price < 0: return await ctx.send(t("pets.sell.invalid_price", lang))
        pet = get_pet_by_id(pet_id)
        if not pet or pet.user_id != ctx.author.id: return await ctx.send(t("pets.sell.not_owned", lang))
        if get_balance(buyer.id) < price: return await ctx.send(t("pets.sell.buyer_no_money", lang, name=buyer.display_name, price=price))

        pet_data = PETS_DB.get(pet.pet_type)
        loc_rarity = get_rarity_name(pet_data['rarity'].name, lang)
        loc_bonus = t(f"bonuses.{pet_data['bonus'].name.lower()}", lang)
        loc_species = get_pet_name(pet.pet_type, lang)

        embed = discord.Embed(title=t("pets.sell.offer_title", lang), color=discord.Color.blue())
        embed.description = t("pets.sell.offer_desc", lang, seller=ctx.author.display_name, buyer=buyer.mention, emoji=pet.emoji, nickname=pet.nickname, id=pet.id, type=loc_species, rarity=loc_rarity, level=pet.level, bonus=loc_bonus, elo=pet.elo, price=price)
        view = PetSellAcceptView(ctx.author, buyer, pet, price, lang)
        msg = await ctx.send(content=t("pets.sell.offer_received", lang, buyer=buyer.mention, seller=ctx.author.mention), embed=embed, view=view)
        view.message = msg
        await view.wait()
        if view.accepted:
            if get_balance(buyer.id) < price: return await ctx.send(t("pets.sell.fail_no_money", lang, name=buyer.display_name))
            update_balance(ctx.author.id, price)
            update_balance(buyer.id, -price)
            transfer_pet(pet.id, buyer.id)
            embed_success = discord.Embed(title=t("pets.sell.success_title", lang), color=discord.Color.green())
            embed_success.description = t("pets.sell.success_desc", lang, buyer=buyer.display_name, emoji=pet.emoji, nickname=pet.nickname, seller=ctx.author.display_name, price=price)
            await ctx.send(embed=embed_success)
        return None

    @commands.command(name='play', aliases=['jouer'])
    async def play_pet(self, ctx):
        """Jouer avec ton familier pour gagner de l'XP."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        server_id = ctx.guild.id if ctx.guild else None
        pet = get_active_pet(ctx.author.id, server_id)
        if not pet:
            return await ctx.send(t("pets.play.no_pet", lang))
        if pet.on_expedition:
            return await ctx.send(t("expedition.pet_on_expedition", lang, name=pet.nickname))
        xp_gain = random.randint(10, 25)
        leveled_up = pet.add_xp(xp_gain)
        update_pet(pet)
        if leveled_up:
            await ctx.send(t("pets.play.level_up", lang, name=pet.nickname, level=pet.level))
        await ctx.send(t("pets.play.success", lang, name=pet.nickname, xp=xp_gain))

    @commands.command(name='feed', aliases=['nourrir'])
    async def feed_pet(self, ctx, *, item_name: str = None):
        """Nourrir ton familier pour booster ses stats ou le reset."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if item_name is None: return await ctx.send(t("pets.feed.no_item", lang))
        item = ITEMS_REGISTRY.get(item_name.lower())
        if not item: return await ctx.send(t("pets.feed.item_not_found", lang, name=item_name))
        server_id = ctx.guild.id if ctx.guild else None
        pet = get_active_pet(ctx.author.id, server_id)
        if not pet: return await ctx.send(t("pets.play.no_pet", lang))
        if pet.on_expedition: return await ctx.send(t("expedition.pet_on_expedition", lang, name=pet.nickname))

        if isinstance(item, ForgetPotion):
            if pet.forget_xp():
                remove_item_from_inventory(ctx.author.id, item.name, 1)
                update_pet(pet)
                await ctx.send(t("pets.feed.forget_success", lang))
            else: await ctx.send(t("pets.feed.forget_fail", lang))
            return

        if not has_item(ctx.author.id, item.name, 1): return await ctx.send(t("pets.feed.no_inventory", lang, name=item.display_name(lang)))
        msg_error = pet.feed(item, lang)
        if msg_error: return await ctx.send(msg_error)
        remove_item_from_inventory(ctx.author.id, item.name, 1)
        update_pet(pet)
        stat_name = t(f"stats.{item.pet_effect['stat']}", lang)
        sign = "+" if item.pet_effect["amount"] >= 0 else ""
        embed = discord.Embed(title=t("pets.feed.miam_title", lang), color=discord.Color.green())
        embed.description = t("pets.feed.miam_desc", lang, item=item.display_name(lang), emoji=pet.emoji, name=pet.nickname, stat=stat_name, sign=sign, amount=item.pet_effect['amount'], food=pet.food_eaten, max_food=pet.max_food_capacity)
        await ctx.send(embed=embed)

    @commands.command(name='heal', aliases=['soigner'])
    async def heal_pet(self, ctx):
        """Soigner ton familier au centre Pokémon (payant)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        server_id = ctx.guild.id if ctx.guild else None
        pet = get_active_pet(ctx.author.id, server_id)
        if not pet: return await ctx.send(t("pets.play.no_pet", lang))
        if pet.on_expedition: return await ctx.send(t("expedition.pet_on_expedition", lang, name=pet.nickname))
        if pet.hp >= pet.max_hp: return await ctx.send(t("pets.heal.full_hp", lang, name=pet.nickname))
        missing_hp = pet.max_hp - pet.hp
        cost = max(1, int(missing_hp * 0.5))
        if get_balance(ctx.author.id) < cost: return await ctx.send(t("pets.heal.no_money", lang, price=cost))
        update_balance(ctx.author.id, -cost)
        pet.heal_full()
        update_pet(pet)
        await ctx.send(t("pets.heal.success", lang, name=pet.nickname, hp=int(missing_hp), price=cost))

    @commands.command(name='petstats', aliases=['pstats', 'pet_stats', 'pet_stat', "ps"])
    async def pet_stats(self, ctx, user: discord.User = None):
        """Voir les statistiques détaillées de ton familier actif."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user = user or ctx.author
        server_id = ctx.guild.id if ctx.guild else None
        pet = get_active_pet(user.id, server_id)
        if not pet: return await ctx.send(t("pets.play.no_pet", lang))
        pet_data = PETS_DB.get(pet.pet_type)
        loc_rarity = get_rarity_name(pet_data['rarity'].name, lang)
        if pet.level >= 5:
            all_pets = get_all_pets(None, server_id)
            all_pets.sort(key=lambda p: p.elo, reverse=True)
            rank = 1
            for p in all_pets:
                if p.id == pet.id: break
                rank += 1
            total_pets = len(all_pets)
            percentile = (rank / total_pets) if total_pets > 0 else 1
            if rank <= 5: elo_str = t("pets.stats.elo_top", lang, elo=pet.elo, rank="TOP 5 👑")
            elif percentile <= 0.25: elo_str = t("pets.stats.elo_rank", lang, elo=pet.elo, rank="Diamant 💎", progress=int((0.25 - percentile) / 0.25 * 100))
            elif percentile <= 0.50: elo_str = t("pets.stats.elo_rank", lang, elo=pet.elo, rank="Or 🥇", progress=int((0.50 - percentile) / 0.25 * 100))
            elif percentile <= 0.75: elo_str = t("pets.stats.elo_rank", lang, elo=pet.elo, rank="Argent 🥈", progress=int((0.75 - percentile) / 0.25 * 100))
            else: elo_str = t("pets.stats.elo_rank", lang, elo=pet.elo, rank="Bronze 🥉", progress=int((1.0 - percentile) / 0.25 * 100))
        else: elo_str = t("pets.stats.elo_locked", lang)
        loc_type = get_pet_name(pet.pet_type, lang)
        loc_bonus = t(f"bonuses.{pet_data['bonus'].name.lower()}", lang)
        embed = discord.Embed(title=t("pets.stats.title", lang, name=pet.nickname, emoji=pet.emoji), color=discord.Color.blue())
        embed.description = t("pets.stats.desc", lang, type=loc_type, level=pet.level, xp=int(pet.xp), bonus=loc_bonus, bonus_val=pet.level * 2, elo=elo_str)
        attrs = ["max_hp", "hp", "atk", "defense", "speed", "dge", "acc", "crit_c", "crit_d", "spc_c", "trs_lvl"]
        attr_vals = {f"{a}_label": t(f"stats.{a}", lang) for a in attrs}
        for a in attrs:
            val = getattr(pet, a)
            if isinstance(val, float): val = round(val, 2)
            attr_vals[a] = val
        embed.add_field(name=t("pets.stats.attr_title", lang), value=t("pets.stats.attr_val", lang, **attr_vals), inline=False)
        embed.set_footer(text=t("pets.stats.footer", lang, food=pet.food_eaten, max_food=pet.max_food_capacity, rarity=loc_rarity))
        await ctx.send(embed=embed)

    @commands.command(name='pet_battle', aliases=['arene', "petbattle", "pb"])
    @commands.cooldown(1, 10, commands.BucketType.user)
    async def pet_battle(self, ctx, opponent: discord.Member, bet: int = 0):
        """Défier un autre joueur dans un combat de familiers."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if opponent.id == ctx.author.id: return await ctx.send(t("pets.battle.wrong_opponent", lang))
        if bet < 0:
            return await ctx.send(t("pets.sell.invalid_price", lang))
        if bet > 0:
            if get_balance(ctx.author.id) < bet:
                return await ctx.send(t("pets.battle.no_money_challenger", lang, bet=bet))
            if get_balance(opponent.id) < bet:
                return await ctx.send(t("pets.battle.no_money_opponent", lang, name=opponent.display_name))
        server_id = ctx.guild.id if ctx.guild else None
        pet1, pet2 = get_active_pet(ctx.author.id, server_id), get_active_pet(opponent.id, server_id)
        if not pet1:
            return await ctx.send(t("pets.play.no_pet", lang))
        if not pet2:
            return await ctx.send(t("pets.battle.opponent_no_pet", lang, name=opponent.display_name))
        if pet1.on_expedition:
            return await ctx.send(t("expedition.pet_on_expedition", lang, name=pet1.nickname))
        if pet2.on_expedition:
            return await ctx.send(t("expedition.pet_on_expedition", lang, name=pet2.nickname))
        if not pet1.is_alive:
            return await ctx.send(t("pets.battle.pet_ko", lang, name=pet1.nickname))
        if not pet2.is_alive:
            return await ctx.send(t("pets.battle.opponent_pet_ko", lang, name=opponent.display_name))
        view = BattleAcceptView(ctx.author, opponent, bet, lang)
        msg_content = t("pets.battle.challenge_msg", lang, challenger=ctx.author.display_name, opponent=opponent.mention, pet=pet1.nickname)
        if bet > 0: msg_content += t("pets.battle.bet_msg", lang, bet=bet)
        msg = await ctx.send(content=msg_content, view=view)
        view.message = msg
        await view.wait()
        if not view.accepted:
            return None
        if bet > 0:
            if get_balance(ctx.author.id) < bet or get_balance(opponent.id) < bet:
                return await ctx.send(t("pets.battle.money_spent_cancel", lang))
            update_balance(ctx.author.id, -bet); update_balance(opponent.id, -bet)
        embed = discord.Embed(title=t("pets.battle.arena_title", lang), color=discord.Color.dark_theme())

        def update_embed_fields():
            embed.clear_fields()
            embed.add_field(name=f"{pet1.emoji} {pet1.nickname} (Niv {pet1.level})", value=t("pets.battle.master_label", lang, name=ctx.author.display_name) + f"\nPV : {generate_hp_bar(pet1.hp, pet1.max_hp)}\n`{int(pet1.hp)} / {pet1.max_hp}`", inline=True)
            embed.add_field(name="VS", value="⚡", inline=True)
            embed.add_field(name=f"{pet2.emoji} {pet2.nickname} (Niv {pet2.level})", value=t("pets.battle.master_label", lang, name=opponent.display_name) + f"\nPV : {generate_hp_bar(pet2.hp, pet2.max_hp)}\n`{int(pet2.hp)} / {pet2.max_hp}`", inline=True)
        update_embed_fields(); embed.description = t("pets.battle.arena_intro", lang)
        await msg.edit(content=None, embed=embed, view=None); await asyncio.sleep(2)
        await simulate_battle(pet1, pet2, msg, embed, update_embed_fields, sleep_time=0.5, send_messages=True, log_size=10, lang=lang)
        update_pet(pet1); update_pet(pet2)
        if pet1.is_alive and not pet2.is_alive:
            winner, win_pet, lose_pet, result = ctx.author, pet1, pet2, 1.0
        elif pet2.is_alive and not pet1.is_alive:
            winner, win_pet, lose_pet, result = opponent, pet2, pet1, 0.0
        else:
            winner, win_pet, lose_pet, result = None, pet1, pet2, 0.5
        diff1, diff2 = pet1.update_elo(pet2, result)
        update_pet(pet1); update_pet(pet2)
        if server_id:
            update_pet_elo(pet1.id, server_id, pet1.elo)
            update_pet_elo(pet2.id, server_id, pet2.elo)
        if winner:
            embed.color = discord.Color.green(); embed.set_footer(text=t("pets.battle.victory_footer", lang, name=winner.display_name.upper()))
            elo_msg = t("pets.battle.elo_update_title", lang)
            elo_msg += t("pets.battle.elo_update_line", lang, emoji=pet1.emoji, name=pet1.nickname, elo=pet1.elo, diff=f"+{diff1}" if result == 1.0 else str(diff1))
            elo_msg += t("pets.battle.elo_update_line", lang, emoji=pet2.emoji, name=pet2.nickname, elo=pet2.elo, diff=f"+{diff2}" if result == 0.0 else str(diff2))
            embed.description += elo_msg
            if bet > 0: update_balance(winner.id, bet * 2); embed.description += t("pets.battle.win_pot_msg", lang, name=winner.display_name, pot=bet * 2)
        else:
            embed.color = discord.Color.orange(); embed.set_footer(text=t("pets.battle.draw_footer", lang)); embed.description += t("pets.battle.draw_msg", lang)
            elo_msg = t("pets.battle.elo_update_title", lang)
            elo_msg += t("pets.battle.elo_update_line", lang, emoji=pet1.emoji, name=pet1.nickname, elo=pet1.elo, diff=str(diff1))
            elo_msg += t("pets.battle.elo_update_line", lang, emoji=pet2.emoji, name=pet2.nickname, elo=pet2.elo, diff=str(diff2))
            embed.description += elo_msg
            if bet > 0: update_balance(ctx.author.id, bet); update_balance(opponent.id, bet); embed.description += t("pets.battle.refund_msg", lang, bet=bet)
        return await msg.edit(embed=embed)

    @commands.command(name='tradeup', aliases=['fusion'])
    async def trade_up(self, ctx, *pet_ids: int):
        """Échanger plusieurs familiers contre un de rareté supérieure."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if not pet_ids: return await ctx.send(t("pets.tradeup.no_ids", lang))
        if len(pet_ids) != len(set(pet_ids)): return await ctx.send(t("pets.tradeup.duplicate_ids", lang))
        selected_pets = []
        for pid in pet_ids:
            p = get_pet_by_id(pid)
            if not p:
                return await ctx.send(t("pets.tradeup.pet_not_found", lang, id=pid))
            if p.user_id != ctx.author.id:
                return await ctx.send(t("pets.tradeup.not_owned", lang, id=pid))
            if p.is_active:
                return await ctx.send(t("pets.tradeup.pet_active", lang, id=pid))
            if p.on_expedition:
                return await ctx.send(t("expedition.pet_on_expedition", lang, name=p.nickname))
            selected_pets.append(p)
        rarity = selected_pets[0].rarity
        if any(p.rarity != rarity for p in selected_pets):
            return await ctx.send(t("pets.tradeup.different_rarity", lang))
        r_map = {ItemRarity.common: (5, ItemRarity.rare), ItemRarity.rare: (4, ItemRarity.epic), ItemRarity.epic: (3, ItemRarity.legendary)}
        if rarity == ItemRarity.legendary:
            return await ctx.send(t("pets.tradeup.legendary_max", lang))
        req_count, target_rarity = r_map[rarity]
        if len(selected_pets) != req_count:
            return await ctx.send(t("pets.tradeup.wrong_count", lang, req=req_count, rarity=get_rarity_name(rarity.name, lang), target=get_rarity_name(target_rarity.name, lang), count=len(selected_pets)))
        for p in selected_pets:
            delete_pet(p.id)
        new_pet_name = self.roll_gacha_pet(target_rarity); new_pet = Pet.create_new(ctx.author.id, new_pet_name); insert_new_pet(new_pet)
        pet_data = PETS_DB[new_pet_name]; loc_new_name, loc_rarity_new, loc_bonus = get_pet_name(new_pet_name, lang), get_rarity_name(target_rarity.name, lang), t(f"bonuses.{pet_data['bonus'].name.lower()}", lang)
        embed = discord.Embed(title=t("pets.tradeup.success_title", lang), color=discord.Color.gold())
        embed.description = t("pets.tradeup.success_desc", lang, emoji=pet_data['emoji'], name=loc_new_name, rarity=loc_rarity_new, bonus=loc_bonus)
        return await ctx.send(embed=embed)

async def setup(bot):
    await bot.add_cog(Pets(bot))