import discord
from discord.ext import commands
import random
import asyncio

from src.database.achievement import check_and_unlock_achievements, format_achievements_unlocks
from src.database.balance import update_balance, get_balance
from src.database.item import has_item, remove_item_from_inventory, get_item_name_by_id
from src.database.pets import get_all_pets, set_active_pet, get_active_pet, insert_new_pet, update_pet, get_pet_by_id, \
    transfer_pet
from src.globals import ITEMS_REGISTRY
from src.items.Item import ItemRarity, Item
from src.items.MysteryEgg import MysteryEgg
from src.models.Pet import PETS_DB, Pet, PetBonus
from src.utils.embed_utils import generate_hp_bar

STAT_DISPLAY = {
    "max_hp": "❤️ PV Max",
    "hp": "❤️ Points de Vie",
    "atk": "⚔️ Attaque",
    "defense": "🛡️ Défense",
    "speed": "⚡ Vitesse",
    "dge": "💨 Esquive (%)",
    "acc": "🎯 Précision (%)",
    "crit_c": "💥 Chance Critique (%)",
    "crit_d": "💥 Dégâts Critiques",
    "spc_c": "⭐ Chance d'effets (%)",
    "trs_lvl": "👑 Niveau de transcendance"
}

DISPLAY_BONUS = {
    PetBonus.FISH: "Aide à pêcher",
    PetBonus.MINE: "Aide à miner",
    PetBonus.FARM: "Améliore la ferme",
    PetBonus.HUNT: "Aide à la chasse",
}


class BattleAcceptView(discord.ui.View):
    def __init__(self, challenger, opponent, bet):
        super().__init__(timeout=60.0)
        self.challenger = challenger
        self.opponent = opponent
        self.bet = bet
        self.accepted = None

    @discord.ui.button(label="Accepter le Duel", style=discord.ButtonStyle.success, emoji="⚔️")
    async def accept(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.opponent.id:
            return await interaction.response.send_message("❌ Ce défi ne t'est pas destiné !", ephemeral=True)

        self.accepted = True
        for child in self.children:
            child.disabled = True

        await interaction.response.edit_message(
            content=f"🔥 **Le défi est accepté !** Préparation de l'arène...",
            view=self
        )
        self.stop()

    @discord.ui.button(label="Refuser", style=discord.ButtonStyle.danger, emoji="🏃")
    async def decline(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.opponent.id:
            return await interaction.response.send_message("❌ Ce défi ne t'est pas destiné !", ephemeral=True)
        self.accepted = False
        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(
            content=f"🏃 **{self.opponent.display_name}** a refusé le combat.",
            view=self
        )
        self.stop()

    async def on_timeout(self):
        self.accepted = False
        for child in self.children:
            child.disabled = True
        try:
            await self.message.edit(content="⏳ **Le défi a expiré.**", view=self)
        except:
            pass


class PetSellAcceptView(discord.ui.View):
    def __init__(self, seller, buyer, pet, price):
        super().__init__(timeout=60.0)
        self.seller = seller
        self.buyer = buyer
        self.pet = pet
        self.price = price
        self.accepted = None

    @discord.ui.button(label="Acheter", style=discord.ButtonStyle.success, emoji="💰")
    async def accept(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.buyer.id:
            return await interaction.response.send_message("❌ Cette offre ne t'est pas destinée !", ephemeral=True)

        if get_balance(self.buyer.id) < self.price:
             return await interaction.response.send_message("❌ Tu n'as pas assez d'argent pour acheter ce familier !", ephemeral=True)

        self.accepted = True
        for child in self.children:
            child.disabled = True

        await interaction.response.edit_message(
            content=f"🎉 **Transaction acceptée !** Transfert du familier en cours...",
            view=self
        )
        self.stop()

    @discord.ui.button(label="Refuser", style=discord.ButtonStyle.danger, emoji="✖️")
    async def decline(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.buyer.id:
            return await interaction.response.send_message("❌ Cette offre ne t'est pas destinée !", ephemeral=True)
        self.accepted = False
        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(
            content=f"✖️ **{self.buyer.display_name}** a refusé l'offre.",
            view=self
        )
        self.stop()

    async def on_timeout(self):
        self.accepted = False
        for child in self.children:
            child.disabled = True
        try:
            await self.message.edit(content="⏳ **L'offre a expiré.**", view=self)
        except:
            pass


class Pets(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    def roll_gacha_pet(self):
        roll = random.random()
        if roll < 0.05:
            target_rarity = ItemRarity.legendary  # 5%
        elif roll < 0.1:
            target_rarity = ItemRarity.epic  # 10%
        elif roll < 0.30:
            target_rarity = ItemRarity.rare  # 30%
        else:
            target_rarity = ItemRarity.common  # 55%

        possible_pets = [name for name, data in PETS_DB.items() if data["rarity"] == target_rarity]
        return random.choice(possible_pets)

    @commands.command(name='hatch', aliases=['eclore'])
    async def hatch(self, ctx):
        """Faire éclore un Œuf Mystère."""
        user_id = ctx.author.id
        me_name = MysteryEgg().name
        if not has_item(user_id, me_name, 1):
            return await ctx.send("❌ Tu n'as pas d'Œuf Mystère dans ton inventaire ! Achètes-en un au `!shop`.")

        remove_item_from_inventory(user_id, me_name, 1)

        embed = discord.Embed(title="🥚 Éclosion en cours...", color=discord.Color.light_grey())
        embed.description = "L'œuf tremble...\n*Crac...*"
        msg = await ctx.send(embed=embed)

        await asyncio.sleep(2)

        embed.description = "La coquille se brise !\n*CRAC CRAC...*"
        await msg.edit(embed=embed)

        await asyncio.sleep(2)

        pet_name = self.roll_gacha_pet()
        pet_data = PETS_DB[pet_name]

        insert_new_pet(Pet.create_new(user_id, pet_name, pet_name))

        color = discord.Color.blue()
        if pet_data["rarity"] == ItemRarity.legendary:
            color = discord.Color.gold()
        elif pet_data["rarity"] == ItemRarity.epic:
            color = discord.Color.purple()

        embed.title = f"{pet_data['emoji']} UN NOUVEAU COMPAGNON !"
        embed.color = color
        embed.description = (
            f"Félicitations ! Tu as obtenu un **{pet_name}** !\n\n"
            f"🌟 **Rareté :** {pet_data['rarity']}\n"
            f"⚡ **Bonus Passif :** {DISPLAY_BONUS[pet_data['bonus']]}\n\n"
            f"Tape `!pets` pour voir ta collection et l'équiper."
        )
        await msg.edit(embed=embed)

    @commands.command(name='pets', aliases=['familiers', 'zoo'])
    async def my_pets(self, ctx):
        """Voir ta collection de familiers."""
        my_pets = get_all_pets(ctx.author.id)
        if not my_pets:
            return await ctx.send("🐾 Tu n'as aucun familier. Achète un Œuf Mystère au shop !")
        embed = discord.Embed(title=f"🐾 La Ménagerie de {ctx.author.name}", color=discord.Color.green())
        desc = ""
        for pet in my_pets:
            status = "🔵 (Actif)" if pet.is_active else "🔴 (Inactif)" + f" ID: `{pet.id}`"
            desc += f"{pet.emoji} **{pet.nickname}** - {status}\n"
        embed.description = desc
        embed.set_footer(text="Pour t'équiper d'un familier : !equip <ID>")
        return await ctx.send(embed=embed)

    @commands.command(name='equip')
    async def equip(self, ctx, pet_id: int):
        """Équiper un familier actif."""
        success = set_active_pet(ctx.author.id, pet_id)
        if success:
            await ctx.send(f"✅ Familier équipé avec succès !")
        else:
            await ctx.send(f"❌ Impossible d'équiper ce familier. Es-tu sûr de le posséder ?")


    @commands.command(name='pet_rename')
    async def rename_pet(self, ctx, *, new_name: str):
        """Renommer ton familier actif."""
        if len(new_name) > 20: return await ctx.send("❌ Ce nom est trop long (Max 20 caractères).")
        pet = get_active_pet(ctx.author.id)
        pet.nickname = new_name
        update_pet(pet)
        await ctx.send(f"🏷️ Ton familier actif s'appelle désormais **{new_name}** !")
        return None

    @commands.command(name='sell_pet', aliases=['vendre_familier'])
    async def sell_pet(self, ctx, pet_id: int, buyer: discord.Member, price: int):
        """Vendre un familier à un autre joueur."""
        if ctx.author.id == buyer.id:
            return await ctx.send("❌ Tu ne peux pas te vendre un familier à toi-même !")
        
        if price < 0:
            return await ctx.send("❌ Le prix doit être positif !")

        pet = get_pet_by_id(pet_id)
        if not pet or pet.user_id != ctx.author.id:
            return await ctx.send("❌ Ce familier ne t'appartient pas ou n'existe pas !")

        if get_balance(buyer.id) < price:
             return await ctx.send(f"❌ {buyer.display_name} n'a pas assez d'argent pour acheter ce familier ({price} $).")

        pet_data = PETS_DB.get(pet.pet_type, {})
        rarity = pet_data.get("rarity", "Standard")
        if hasattr(rarity, 'value'):
            rarity = rarity.value

        embed = discord.Embed(title="📜 Offre de Vente de Familier", color=discord.Color.gold())
        embed.description = (
            f"**{ctx.author.display_name}** propose de vendre son familier à {buyer.mention} :\n\n"
            f"{pet.emoji} **{pet.nickname}** (ID: `{pet.id}`)\n"
            f"**Espèce :** {pet.pet_type}\n"
            f"**Rareté :** {rarity}\n"
            f"**Niveau :** {pet.level}\n"
            f"**Niveau :** {DISPLAY_BONUS[pet.bonus]}\n"
            f"**ELO :** {pet.elo}\n\n"
            f"💰 **Prix demandé : {price} $**"
        )

        view = PetSellAcceptView(ctx.author, buyer, pet, price)
        msg_content = f"{buyer.mention}, tu as reçu une offre de {ctx.author.display_name} !"
        msg = await ctx.send(content=msg_content, embed=embed, view=view)
        view.message = msg

        await view.wait()

        if not view.accepted:
            return

        if get_balance(buyer.id) < price:
            await msg.edit(content=f"❌ La transaction a échoué : {buyer.display_name} n'a plus assez d'argent.", embed=None, view=None)
            return
        update_balance(buyer.id, -price)
        update_balance(ctx.author.id, price)
        transfer_pet(pet.id, buyer.id)

        success_embed = discord.Embed(title="🤝 Transaction Réussie !", color=discord.Color.green())
        success_embed.description = (
            f"**{buyer.display_name}** a acheté {pet.emoji} **{pet.nickname}** à **{ctx.author.display_name}** pour **{price} $**.\n"
            f"Prends en bien soin !"
        )
        await msg.edit(content=None, embed=success_embed, view=None)




    @commands.command(name='play')
    @commands.cooldown(1, 7200, commands.BucketType.user)
    async def play_pet(self, ctx):
        """Jouer avec ton familier pour lui donner de l'XP."""
        user_id = ctx.author.id
        pet = get_active_pet(user_id)
        if not pet: return await ctx.send("❌ Tu n'as pas de familier actif !")
        xp_gain = random.randint(35, 100)
        leveled_up = pet.add_xp(xp_gain)
        update_pet(pet)
        if leveled_up:
            await ctx.send(f"🎉 **NIVEAU SUPÉRIEUR !** {pet.nickname} passe au niveau {pet.level} et gagne des stats !")
        else:
            await ctx.send(f"🎾 Tu as joué avec **{pet.nickname}**. Il gagne +{xp_gain} XP.")

        unlocks = check_and_unlock_achievements(int(user_id))
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks))
        return None

    @commands.command(name='feed', aliases=['nourrir', 'booster'])
    async def feed_pet(self, ctx, *, item_name: str = None):
        """Nourrir ton familier pour booster ses stats."""
        if not item_name:
            return await ctx.send("❌ Tu dois préciser ce que tu veux lui donner. Exemple : `!feed Saumon`")
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send("❌ Tu n'as pas de familier actif ! Utilise `!equip`.")

        item_name = item_name.strip()
        if item_name.isdigit():
            resolved = get_item_name_by_id(int(item_name))
            if resolved:
                item_name = resolved
        item: Item = ITEMS_REGISTRY.get(item_name)
        if not item:
            return await ctx.send(f"❌ L'objet **{item_name}** n'existe pas.")

        error_msg = pet.feed(item)
        if error_msg:
            return await ctx.send(error_msg)
        if not has_item(ctx.author.id, item.name, 1):
            return await ctx.send(f"❌ Tu n'as pas de **{item.name}** dans ton inventaire !")

        remove_item_from_inventory(ctx.author.id, item.name, 1)
        update_pet(pet)
        stat_to_boost, boost_amount = item.pet_effect['stat'], item.pet_effect['amount']
        stat_name_clean = STAT_DISPLAY.get(stat_to_boost, stat_to_boost.upper())
        sign = "+" if boost_amount > 0 else ""

        embed = discord.Embed(title="🍖 Miam !", color=discord.Color.green())
        embed.description = (
            f"Tu as donné **1x {item.name}** à {pet.emoji} **{pet.nickname}**.\n\n"
            f"✨ **EFFET :** {stat_name_clean} {sign}{boost_amount}\n\n"
            f"📊 *Estomac : {pet.food_eaten} / {pet.max_food_capacity}*"
        )
        return await ctx.send(embed=embed)



    @commands.command(name='heal', aliases=['soin', "pet_heal"])
    async def heal_pet(self, ctx):
        """Soigner ton familier."""
        user_id = int(ctx.author.id)
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send("❌ Tu n'as pas de familier actif !")
        if pet.hp >= pet.max_hp:
            return await ctx.send(f"✨ **{pet.nickname}** est déjà en pleine forme !")
        
        restored = pet.max_hp - pet.hp
        pet.hp = pet.max_hp
        bal = get_balance(user_id)
        if pet.level < 5:
            price = 0
        elif pet.level < 10:
            price = restored
        elif pet.level < 15:
            price = restored * 5
        else:
            price = restored * 10
        if bal < price:
            await ctx.send(f"❌ Tu n'as pas assez d'argent pour payer les soins prix: {price}$")
            return None
        update_balance(user_id, -price)
        update_pet(pet)
        return await ctx.send(f"🏥 **{pet.nickname}** s'est bien reposé et a récupéré **{int(restored)} PV** ! Ses PV sont maintenant au maximum.")

    @commands.command(name='petstats', aliases=['pstats', 'pet_stats', 'pet_stat'])
    async def pet_stats(self, ctx):
        """Voir les statistiques de ton familier actif."""
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send("❌ Tu n'as pas de familier actif !")
        
        pet_data = PETS_DB.get(pet.pet_type, {})
        rarity = pet_data.get("rarity", "Standard")
        if hasattr(rarity, 'value'):
            rarity = rarity.value
            
        embed = discord.Embed(title=f"📊 Stats de {pet.nickname} {pet.emoji}", color=discord.Color.teal())
        embed.description = (
            f"**Type:** {pet.pet_type.capitalize()} | **Niveau:** {pet.level} | **XP:** {pet.xp}\n"
            f"**Bonus:** {DISPLAY_BONUS[pet.bonus]} ({pet.level // 4}%)\n"
            f"**ELO (Classement):** {pet.elo} 🏆"
        )
        
        stats_str = (
            f"**{STAT_DISPLAY['hp']} :** `{int(pet.hp)} / {pet.max_hp}`\n"
            f"**{STAT_DISPLAY['atk']} :** `{pet.atk}`\n"
            f"**{STAT_DISPLAY['defense']} :** `{pet.defense}`\n"
            f"**{STAT_DISPLAY['speed']} :** `{pet.speed}`\n"
            f"**{STAT_DISPLAY['dge']} :** `{pet.dge}%`\n"
            f"**{STAT_DISPLAY['acc']} :** `{pet.acc}%`\n"
            f"**{STAT_DISPLAY['crit_c']} :** `{pet.crit_c}%`\n"
            f"**{STAT_DISPLAY['crit_d']} :** `{pet.crit_d}x`\n"
            f"**{STAT_DISPLAY['spc_c']} :** `{pet.spc_c}x`\n"
            f"**{STAT_DISPLAY['trs_lvl']} :** `{pet.trs_lvl}x`\n"
        )
        embed.add_field(name="Attributs de Combat", value=stats_str, inline=False)
        embed.set_footer(text=f"Estomac : {pet.food_eaten} / {pet.max_food_capacity} | Rareté : {rarity}")
        return await ctx.send(embed=embed)

    @commands.command(name='pet_battle', aliases=['arene', "petbattle"])
    @commands.cooldown(1, 10, commands.BucketType.user)
    async def pet_battle(self, ctx, opponent: discord.Member, bet: int = 0):
        """Défier un autre joueur dans un combat de familiers."""
        if opponent.id == ctx.author.id:
            return await ctx.send("❌ Tu ne peux pas t'affronter toi-même !")
        if bet < 0:
            return await ctx.send("❌ La mise ne peut pas être négative.")

        if bet > 0:
            if get_balance(ctx.author.id) < bet:
                return await ctx.send(f"❌ Tu n'as pas assez d'argent pour miser **{bet} $**.")
            if get_balance(opponent.id) < bet:
                return await ctx.send(
                    f"❌ {opponent.display_name} n'a pas les fonds nécessaires pour suivre cette mise.")

        pet1 = get_active_pet(ctx.author.id)
        pet2 = get_active_pet(opponent.id)

        if not pet1: return await ctx.send("❌ Tu n'as pas de familier actif !")
        if not pet2: return await ctx.send(f"❌ {opponent.display_name} n'a pas de familier actif !")
        if not pet1.is_alive: return await ctx.send(f"❌ Ton familier **{pet1.nickname}** est K.O. ! Utilise `!feed`.")
        if not pet2.is_alive: return await ctx.send(f"❌ Le familier de {opponent.display_name} est K.O. !")

        view = BattleAcceptView(ctx.author, opponent, bet)

        msg_content = f"⚔️ {opponent.mention}, **{ctx.author.display_name}** te défie en duel avec son **{pet1.nickname}** !"
        if bet > 0:
            msg_content += f"\n💰 **Mise en jeu : {bet} $** (Le gagnant remporte tout !)"
        msg = await ctx.send(content=msg_content, view=view)
        view.message = msg
        await view.wait()
        if not view.accepted:
            return

        if bet > 0:
            if get_balance(ctx.author.id) < bet or get_balance(opponent.id) < bet:
                return await ctx.send("❌ L'un des joueurs a dépensé son argent avant le début du combat. Annulation !")

            update_balance(ctx.author.id, -bet)
            update_balance(opponent.id, -bet)

        embed = discord.Embed(title="⚔️ ARÈNE DES FAMILIERS ⚔️", color=discord.Color.dark_theme())

        def update_embed_fields():
            embed.clear_fields()
            embed.add_field(
                name=f"{pet1.emoji} {pet1.nickname} (Niv {pet1.level})",
                value=f"Maître : {ctx.author.display_name}\nPV : {generate_hp_bar(pet1.hp, pet1.max_hp)}\n`{int(pet1.hp)} / {pet1.max_hp}`",
                inline=True
            )
            embed.add_field(name="VS", value="⚡", inline=True)
            embed.add_field(
                name=f"{pet2.emoji} {pet2.nickname} (Niv {pet2.level})",
                value=f"Maître : {opponent.display_name}\nPV : {generate_hp_bar(pet2.hp, pet2.max_hp)}\n`{int(pet2.hp)} / {pet2.max_hp}`",
                inline=True
            )

        update_embed_fields()
        embed.description = "🔥 *Le public retient son souffle...*"
        await msg.edit(content=None, embed=embed, view=None)
        await asyncio.sleep(2)


        log = []
        turn = 1
        fighters = [pet1, pet2] if pet1.speed >= pet2.speed else [pet2, pet1]

        while pet1.is_alive and pet2.is_alive and turn <= 35:
            for i in range(2):
                attacker = fighters[i]
                defender = fighters[1 - i]

                if not attacker.is_alive: continue

                action_text = attacker.attack(defender)
                log.append(action_text)

                if len(log) > 10: log.pop(0)

                update_embed_fields()
                embed.description = "📜 **Journal de combat :**\n\n" + "\n".join(log)
                await msg.edit(embed=embed)
                await asyncio.sleep(0.5)

                if not defender.is_alive: break
            turn += 1

        update_pet(pet1)
        update_pet(pet2)

        if pet1.is_alive and not pet2.is_alive:
            result = 1.0
            winner = ctx.author
            loser_pet = pet2.nickname
        elif pet2.is_alive and not pet1.is_alive:
            result = 0.0
            winner = opponent
            loser_pet = pet1.nickname
        else:
            result = 0.5
            winner = None

        diff1, diff2 = pet1.update_elo(pet2, result)

        update_pet(pet1)
        update_pet(pet2)

        if winner:
            embed.color = discord.Color.green()
            embed.set_footer(text=f"🏆 VICTOIRE DE {winner.display_name.upper()} !")
            embed.description += f"\n\n💀 **{loser_pet} est K.O. !**\n"

            embed.description += f"\n📊 **Mise à jour du Classement (ELO) :**"
            embed.description += f"\n📈 {pet1.nickname} : **{pet1.elo}** ({diff1})"
            embed.description += f"\n📉 {pet2.nickname} : **{pet2.elo}** ({diff2})"
            if bet > 0:
                pot = bet * 2
                update_balance(winner.id, pot)
                embed.description += f"\n💰 **{winner.display_name} remporte le pot de {pot} $ !**"
        else:
            embed.color = discord.Color.orange()
            embed.set_footer(text="🤝 MATCH NUL ! Limite de tours atteinte.")
            embed.description += "\n\n💨 *Les deux familiers s'effondrent de fatigue...*\n"

            embed.description += f"\n📊 **Mise à jour du Classement (ELO) :**"
            embed.description += f"\n⚖️ {pet1.nickname} : **{pet1.elo}** ({diff1})"
            embed.description += f"\n⚖️ {pet2.nickname} : **{pet2.elo}** ({diff2})"
            if bet > 0:
                update_balance(ctx.author.id, bet)
                update_balance(opponent.id, bet)
                embed.description += f"\n💰 Les mises ({bet} $) ont été remboursées."
        update_embed_fields()
        await msg.edit(embed=embed)
        if winner:
            embed.color = discord.Color.green()
            embed.set_footer(text=f"🏆 VICTOIRE DE {winner.display_name.upper()} !")
            embed.description += f"\n\n💀 **{loser_pet} est K.O. !**"
            if bet > 0:
                pot = bet * 2
                update_balance(winner.id, pot)
                embed.description += f"\n💰 **{winner.display_name} remporte le pot de {pot} $ !**"
        else:
            embed.color = discord.Color.orange()
            embed.set_footer(text="🤝 MATCH NUL ! Limite de tours atteinte.")
            embed.description += "\n\n💨 *Les deux familiers s'effondrent de fatigue...*"
            if bet > 0:
                update_balance(ctx.author.id, bet)
                update_balance(opponent.id, bet)
                embed.description += f"\n💰 Les mises ({bet} $) ont été remboursées."
        update_embed_fields()
        await msg.edit(embed=embed)

async def setup(bot):
    await bot.add_cog(Pets(bot))