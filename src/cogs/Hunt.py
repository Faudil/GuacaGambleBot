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
from src.utils.battle import simulate_battle
from src.database.settings import get_language
from src.utils.i18n import t, get_item_name

HUNT_ZONES = {
    "easy": {
        "key": "easy_zone", "emoji": "🌲", "color": discord.Color.green(),
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
        "key": "medium_zone", "emoji": "🦇", "color": discord.Color.orange(),
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
        "key": "hard_zone", "emoji": "🌋", "color": discord.Color.red(),
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


def generate_enemy(zone_key: str, lang: str) -> Pet:
    zone = HUNT_ZONES[zone_key]
    template = random.choice(zone["enemies"])
    enemy_lvl = random.randint(*zone["level_range"])

    translated_name = t(f"hunt.enemies.{template['name']}", lang)
    nickname = translated_name + t("hunt.wild_suffix", lang)

    enemy = Pet(
        pet_type=template["name"],
        nickname=nickname,
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
    def __init__(self, cog, ctx, pet, lang):
        super().__init__(timeout=30.0)
        self.cog = cog
        self.ctx = ctx
        self.pet = pet
        self.lang = lang

    async def start_hunt(self, interaction: discord.Interaction, zone_key: str):
        if interaction.user.id != self.ctx.author.id:
            return await interaction.response.send_message(t("hunt.wrong_expedition", self.lang), ephemeral=True)

        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(content=t("hunt.generating", self.lang), view=self)

        await self.cog.execute_combat(interaction.message, self.ctx, self.pet, zone_key, self.lang)

    @discord.ui.button(label="easy_label", style=discord.ButtonStyle.success, emoji="🌲")
    async def btn_easy(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.start_hunt(interaction, "easy")

    @discord.ui.button(label="medium_label", style=discord.ButtonStyle.primary, emoji="🦇")
    async def btn_medium(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.start_hunt(interaction, "medium")

    @discord.ui.button(label="hard_label", style=discord.ButtonStyle.danger, emoji="🌋")
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
        lang = get_language(ctx.guild.id if ctx.guild else None)
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send(t("hunt.no_pet", lang))
        if pet.on_expedition:
            return await ctx.send(t("expedition.pet_on_expedition", lang, name=pet.nickname))
        if not pet.is_alive:
            return await ctx.send(t("hunt.pet_ko", lang, name=pet.nickname))

        embed = discord.Embed(title=t("hunt.dashboard_title", lang), color=discord.Color.blue())
        embed.description = t("hunt.dashboard_desc", lang, name=pet.nickname, lvl=pet.level)
        
        for key, zone in HUNT_ZONES.items():
            name = t(f"hunt.{zone['key']}", lang)
            range_str = t("hunt.level_range", lang, min=zone['level_range'][0], max=zone['level_range'][1])
            embed.add_field(name=f"{zone['emoji']} {name}", value=range_str, inline=True)

        view = DifficultyView(self, ctx, pet, lang)
        for item in view.children:
            if isinstance(item, discord.ui.Button):
                if item.label == "easy_label": item.label = t("hunt.easy_label", lang)
                if item.label == "medium_label": item.label = t("hunt.medium_label", lang)
                if item.label == "hard_label": item.label = t("hunt.hard_label", lang)

        await ctx.send(embed=embed, view=view)

    async def execute_combat(self, message: discord.Message, ctx, pet: Pet, zone_key: str, lang: str):
        zone = HUNT_ZONES[zone_key]
        enemy = generate_enemy(zone_key, lang)
        enemy_emoji = getattr(enemy, '_wild_emoji', '👹')
        
        zone_name = t(f"hunt.{zone['key']}", lang)
        embed = discord.Embed(title=t("hunt.expedition_title", lang, emoji=zone['emoji'], name=zone_name), color=zone["color"])

        def update_embed():
            embed.clear_fields()
            embed.add_field(
                name=f"{pet.emoji} {pet.nickname} (Niv {pet.level})",
                value=f"PV : {generate_hp_bar(pet.hp, pet.max_hp)}\n`{int(pet.hp)} / {pet.max_hp}`",
                inline=True
            )
            embed.add_field(name=t("hunt.vs", lang), value="⚡", inline=True)
            embed.add_field(
                name=f"{enemy_emoji} {enemy.nickname} (Niv {enemy.level})",
                value=f"PV : {generate_hp_bar(enemy.hp, enemy.max_hp)}\n`{int(enemy.hp)} / {enemy.max_hp}`",
                inline=True
            )

        update_embed()
        embed.description = t("hunt.enemy_spawn", lang, name=enemy.nickname)
        await message.edit(content=None, embed=embed, view=None)
        await asyncio.sleep(2)

        await simulate_battle(
            pet, enemy, message, embed, update_embed,
            sleep_time=1.5, send_messages=True, log_size=5,
            journal_title=t("hunt.battle_journal", lang),
            lang=lang
        )

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
                    add_item_to_inventory(user_id, item_name.lower(), qty)
                    # Use localized name for loot display
                    looted_items.append(t("farm.harvest_msg", lang, qty=qty, item=get_item_name(item_name, lang)))

            if looted_items:
                embed.description += t("hunt.loot_found", lang) + ", ".join(looted_items)
            else:
                embed.description += t("hunt.no_loot", lang)

            embed.color = discord.Color.gold()
            embed.set_footer(text=t("hunt.victory_footer", lang))
            embed.description += t("hunt.victory_msg", lang, pet=pet.nickname, xp=xp_gained)

        elif enemy.is_alive and not pet.is_alive:
            base_xp = enemy.level * random.randint(15, 25) / 10
            xp_gained = int(base_xp)
            leveled_up = pet.add_xp(xp_gained)
            embed.color = discord.Color.red()
            embed.set_footer(text=t("hunt.defeat_footer", lang))
            embed.description += t("hunt.defeat_msg", lang, pet=pet.nickname, xp=xp_gained)
        else:
            base_xp = enemy.level * random.randint(15, 25) / 2
            xp_gained = int(base_xp * zone["xp_mult"])
            leveled_up = pet.add_xp(xp_gained)
            embed.color = discord.Color.orange()
            embed.set_footer(text=t("hunt.end_footer", lang))
            embed.description += t("hunt.flee_msg", lang, xp=xp_gained)

        if leveled_up:
                embed.description += t("hunt.level_up", lang, pet=pet.nickname, level=pet.level)
        update_pet(pet)
        update_embed()
        await message.edit(embed=embed)
        user_id = message.interaction_metadata.user.id if message.interaction_metadata else ctx.author.id
        unlocks = check_and_unlock_achievements(user_id)
        if unlocks:
            await message.channel.send(content=f"<@{user_id}>", embed=format_achievements_unlocks(unlocks, lang))


async def setup(bot):
    await bot.add_cog(Hunt(bot))