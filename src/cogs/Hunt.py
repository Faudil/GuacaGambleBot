import discord
from discord.ext import commands
import random
import asyncio

from src.command_decorators import daily_limit
from src.database.item import add_item_to_inventory
from src.database.pets import get_active_pet, update_pet
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.items.MysteryEgg import MysteryEgg
from src.models.Pet import Pet, PetBonus
from src.utils.embed_utils import generate_hp_bar

HUNT_ZONES = {
    "easy": {
        "name": "Forêt Paisible", "emoji": "🌲", "color": discord.Color.green(),
        "level_range": (1, 5), "xp_mult": 1.0,
        "enemies": [
            {"name": "Slime Gluant", "emoji": "💧", "hp": 25, "atk": 5, "def": 2, "spd": 5, "dge": 0, "acc": 5},
            {"name": "Sanglier Sauvage", "emoji": "🐗", "hp": 35, "atk": 8, "def": 5, "spd": 10, "dge": 2, "acc": 5}
        ],
        "loot_table": [
            {"item": "Caillou", "chance": 0.50, "max_qty": 3},
            {"item": "Tomate", "chance": 0.30, "max_qty": 2},
            {"item": "Charbon", "chance": 0.15, "max_qty": 1},
            {"item": MysteryEgg().name, "chance": 0.01, "max_qty": 1}
        ]
    },
    "medium": {
        "name": "Grotte Sombre", "emoji": "🦇", "color": discord.Color.orange(),
        "level_range": (8, 12), "xp_mult": 2.5,
        "enemies": [
            {"name": "Gobelin Voleur", "emoji": "👺", "hp": 40, "atk": 18, "def": 5, "spd": 25, "dge": 15, "acc": 10},
            {"name": "Araignée Géante", "emoji": "🕷️", "hp": 50, "atk": 15, "def": 8, "spd": 30, "dge": 10, "acc": 15}
        ],
        "loot_table": [
            {"item": "Charbon", "chance": 0.60, "max_qty": 3},
            {"item": "Minerai de Fer", "chance": 0.40, "max_qty": 2},
            {"item": "Sardine", "chance": 0.20, "max_qty": 1},
            {"item": MysteryEgg().name, "chance": 0.05, "max_qty": 1}
        ]
    },
    "hard": {
        "name": "Sommet du Volcan", "emoji": "🌋", "color": discord.Color.red(),
        "level_range": (15, 20), "xp_mult": 5.0,
        "enemies": [
            {"name": "Golem de Magma", "emoji": "🗿", "hp": 100, "atk": 25, "def": 20, "spd": 2, "dge": 0, "acc": 5},
            {"name": "Drake de Feu", "emoji": "🐉", "hp": 80, "atk": 35, "def": 12, "spd": 25, "dge": 10, "acc": 20}
        ],
        "loot_table": [
            {"item": "Minerai de Cuivre", "chance": 0.50, "max_qty": 5},
            {"item": "Pépite d'Or", "chance": 0.3, "max_qty": 3},
            {"item": "Diamant Brut", "chance": 0.2, "max_qty": 2},
            {"item": MysteryEgg().name, "chance": 0.1, "max_qty": 1}
        ]
    }
}


def generate_enemy(zone_key: str) -> Pet:
    zone = HUNT_ZONES[zone_key]
    template = random.choice(zone["enemies"])
    enemy_lvl = random.randint(*zone["level_range"])

    enemy = Pet(
        pet_type=template["name"],
        nickname=f"{template['name']} Sauvage",
        level=enemy_lvl,
        max_hp=template["hp"] + (enemy_lvl * 5),
        hp=template["hp"] + (enemy_lvl * 5),
        atk=template["atk"] + (enemy_lvl * 2),
        defense=template["def"] + (enemy_lvl * 1),
        speed=template["spd"] + (enemy_lvl * 1),
        dge=template["dge"],
        acc=template["acc"] + (enemy_lvl // 2),
        crit_c=5,
        crit_d=1.5
    )
    enemy._wild_emoji = template["emoji"]
    return enemy


class DifficultyView(discord.ui.View):
    def __init__(self, cog, ctx, pet):
        super().__init__(timeout=30.0)
        self.cog = cog
        self.ctx = ctx
        self.pet = pet

    async def start_hunt(self, interaction: discord.Interaction, zone_key: str):
        if interaction.user.id != self.ctx.author.id:
            return await interaction.response.send_message("❌ Ce n'est pas ton expédition !", ephemeral=True)

        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(content="⚔️ **Génération de l'expédition...**", view=self)

        await self.cog.execute_combat(interaction.message, self.ctx, self.pet, zone_key)

    @discord.ui.button(label="Forêt (Facile)", style=discord.ButtonStyle.success, emoji="🌲")
    async def btn_easy(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.start_hunt(interaction, "easy")

    @discord.ui.button(label="Grotte (Moyen)", style=discord.ButtonStyle.primary, emoji="🦇")
    async def btn_medium(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.start_hunt(interaction, "medium")

    @discord.ui.button(label="Volcan (Difficile)", style=discord.ButtonStyle.danger, emoji="🌋")
    async def btn_hard(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.start_hunt(interaction, "hard")


class Hunt(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='hunt', aliases=['chasse'])
    @commands.cooldown(1, 10, commands.BucketType.user)
    @daily_limit('hunt', 10)
    async def hunt(self, ctx):
        """Envoyer son familier en expédition."""
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send("❌ Tu n'as pas de familier actif ! (`!equip`)")
        if not pet.is_alive:
            return await ctx.send(f"❌ **{pet.nickname}** est K.O. ! Nourris-le pour le soigner avant de repartir.")

        embed = discord.Embed(title="🗺️ Tableau des Expéditions", color=discord.Color.blue())
        embed.description = (
            f"Ton familier **{pet.nickname}** (Niv {pet.level}) est prêt.\n"
            "Choisis la zone que tu souhaites explorer. Plus la zone est difficile, plus tu gagneras d'XP !"
        )
        embed.add_field(name="🌲 Forêt Paisible", value="Niv. 1 à 5", inline=True)
        embed.add_field(name="🦇 Grotte Sombre", value="Niv. 8 à 12", inline=True)
        embed.add_field(name="🌋 Volcan", value="Niv. 15 à 20", inline=True)

        view = DifficultyView(self, ctx, pet)
        await ctx.send(embed=embed, view=view)

    async def execute_combat(self, message: discord.Message, ctx, pet: Pet, zone_key: str):
        zone = HUNT_ZONES[zone_key]
        enemy = generate_enemy(zone_key)
        enemy_emoji = getattr(enemy, '_wild_emoji', '👹')

        embed = discord.Embed(title=f"{zone['emoji']} Expédition : {zone['name']}", color=zone["color"])

        def update_embed():
            embed.clear_fields()
            embed.add_field(
                name=f"{pet.emoji} {pet.nickname} (Niv {pet.level})",
                value=f"PV : {generate_hp_bar(pet.hp, pet.max_hp)}\n`{int(pet.hp)} / {pet.max_hp}`",
                inline=True
            )
            embed.add_field(name="VS", value="⚡", inline=True)
            embed.add_field(
                name=f"{enemy_emoji} {enemy.nickname} (Niv {enemy.level})",
                value=f"PV : {generate_hp_bar(enemy.hp, enemy.max_hp)}\n`{int(enemy.hp)} / {enemy.max_hp}`",
                inline=True
            )

        update_embed()
        embed.description = f"Un **{enemy.nickname}** surgit devant toi !"
        await message.edit(content=None, embed=embed, view=None)
        await asyncio.sleep(2)

        log = []
        turn = 1
        fighters = [pet, enemy] if pet.speed >= enemy.speed else [enemy, pet]

        while pet.is_alive and enemy.is_alive and turn <= 35:
            for i in range(2):
                attacker = fighters[i]
                defender = fighters[1 - i]

                if not attacker.is_alive: continue

                action_text = attacker.attack(defender)
                log.append(action_text)
                if len(log) > 5:
                    log.pop(0)
                update_embed()
                embed.description = "📜 **Combat en cours :**\n\n" + "\n".join(log)
                await message.edit(embed=embed)
                await asyncio.sleep(1.5)

                if not defender.is_alive: break
            turn += 1

        leveled_up = False
        if pet.is_alive and not enemy.is_alive:
            base_xp = enemy.level * random.randint(15, 25)
            xp_gained = int(base_xp * zone["xp_mult"])
            user_id = message.interaction_metadata.user.id if message.interaction_metadata else ctx.author.id
            increment_stat(user_id, "pve_wins")

            leveled_up = pet.add_xp(xp_gained)

            looted_items = []
            bonus = pet.level if pet.bonus == PetBonus.HUNT else 0
            for loot in zone["loot_table"]:
                if random.random() < loot["chance"] + (bonus * 0.01):
                    qty = random.randint(1, loot["max_qty"])
                    item_name = loot["item"]
                    add_item_to_inventory(user_id, item_name, qty)
                    looted_items.append(f"**{qty}x {item_name}**")

            if looted_items:
                embed.description += f"\n\n🎁 **Butin obtenu :**\n" + ", ".join(looted_items)
            else:
                embed.description += f"\n\n🎁 *Le monstre n'a rien laissé derrière lui...*"

            embed.color = discord.Color.gold()
            embed.set_footer(text="🏆 VICTOIRE !")
            embed.description += f"\n\n✨ **{pet.nickname}** a vaincu le monstre et gagne **+{xp_gained} XP** !"

        elif enemy.is_alive and not pet.is_alive:
            base_xp = enemy.level * random.randint(15, 25) / 10
            xp_gained = int(base_xp)
            leveled_up = pet.add_xp(xp_gained)
            embed.color = discord.Color.red()
            embed.set_footer(text="💀 DÉFAITE...")
            embed.description += f"\n\n🩸 **{pet.nickname}** est gravement blessé et fuit le combat... Tu gagnes quand même {xp_gained} XP"
        else:
            base_xp = enemy.level * random.randint(15, 25) / 2
            xp_gained = int(base_xp * zone["xp_mult"])
            leveled_up = pet.add_xp(xp_gained)
            embed.color = discord.Color.orange()
            embed.set_footer(text="🏃 FIN DU COMBAT")
            embed.description += f"\n\n💨 Le monstre s'est enfui dans la nature... Tu gagnes {xp_gained} XP !"

        if leveled_up:
                embed.description += f"\n🎉 **NIVEAU SUPÉRIEUR !** {pet.nickname} passe au Niveau {pet.level} !"
        update_pet(pet)
        update_embed()
        await message.edit(embed=embed)
        user_id = message.interaction_metadata.user.id if message.interaction_metadata else ctx.author.id
        unlocks = check_and_unlock_achievements(user_id)
        if unlocks:
            await message.channel.send(content=f"<@{user_id}>", embed=format_achievements_unlocks(unlocks))


async def setup(bot):
    await bot.add_cog(Hunt(bot))