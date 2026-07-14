import os
import discord
from discord.ext import commands
import asyncio
from src.database.achievement import check_and_unlock_achievements, format_achievements_unlocks
from src.database.balance import update_balance
from src.database.item import add_item_to_inventory
from src.database.pets import get_active_pet, update_pet
from src.database.boss import get_user_stage, set_user_stage
from src.database.settings import get_language
from src.models.Pet import Pet, PETS_DB
from src.utils.battle import simulate_battle
from src.utils.embed_utils import generate_hp_bar
from src.utils.i18n import t, get_pet_name

# Base path for images
COG_DIR = os.path.dirname(os.path.abspath(__file__))
CHAMPION_IMAGE_PATH = os.path.join(COG_DIR, "../../assets/bosses/champion.png")

# Boss stages configuration
BOSS_LEAGUE = [
    {
        "stage": 1,
        "name": "Dresseur Novice",
        "name_en": "Rookie Collector",
        "species": "Souris",
        "level": 5,
        "hp": 80,
        "atk": 15,
        "defense": 8,
        "speed": 15,
        "desc": "Un débutant enthousiaste avec une Souris rapide. Un bon test de départ !",
        "desc_en": "An enthusiastic beginner with a quick Mouse. A good starting test!",
        "reward_money": 200,
        "reward_items": {"café": 1, "œuf mystère": 1},
        "achievement": "boss_league_1",
        "image_path": os.path.join(COG_DIR, "../../assets/bosses/stage_1.png")
    },
    {
        "stage": 2,
        "name": "Gardien de Pierre",
        "name_en": "Stone Sentinel",
        "species": "Ours",
        "level": 10,
        "hp": 150,
        "atk": 25,
        "defense": 20,
        "speed": 8,
        "desc": "Un Ours robuste doté d'une défense impressionnante. Vous devrez percer sa carapace.",
        "desc_en": "A sturdy Bear with impressive defense. You'll need to break through its guard.",
        "reward_money": 500,
        "reward_items": {"ticket vip": 2, "terrain : potager": 1},
        "achievement": "boss_league_2",
        "image_path": os.path.join(COG_DIR, "../../assets/bosses/stage_2.png")
    },
    {
        "stage": 3,
        "name": "Foudre Céleste",
        "name_en": "Storm Striker",
        "species": "Aigle",
        "level": 15,
        "hp": 200,
        "atk": 35,
        "defense": 15,
        "speed": 35,
        "desc": "Un Aigle féroce qui attaque à une vitesse fulgurante et inflige de lourds dégâts critiques.",
        "desc_en": "A fierce Eagle attacking at lightning speed and inflicting heavy critical damage.",
        "reward_money": 1000,
        "reward_items": {"terrain : verger enchanté": 1, "fortune cookie": 2},
        "achievement": "boss_league_3",
        "image_path": os.path.join(COG_DIR, "../../assets/bosses/stage_3.png")
    },
    {
        "stage": 4,
        "name": "Léviathan des Abysses",
        "name_en": "Abyssal Leviathan",
        "species": "Kraken",
        "level": 20,
        "hp": 300,
        "atk": 40,
        "defense": 30,
        "speed": 22,
        "desc": "Le Kraken mythique des profondeurs. Ses attaques de type POISON affaibliront votre familier sur la durée.",
        "desc_en": "The mythical Kraken of the deep. Its POISON-type attacks will wear down your pet over time.",
        "reward_money": 2500,
        "reward_items": {"terrain : serre tropicale": 1, "potion d'oubli": 1},
        "achievement": "boss_league_4",
        "image_path": os.path.join(COG_DIR, "../../assets/bosses/stage_4.png")
    },
    {
        "stage": 5,
        "name": "Le Phénix Éternel",
        "name_en": "The Eternal Phoenix",
        "species": "Phoenix",
        "level": 30,
        "hp": 500,
        "atk": 60,
        "defense": 40,
        "speed": 40,
        "desc": "L'ultime boss de la ligue. Le Phénix renaît de ses cendres avec des stats colossales et des attaques de FEU.",
        "desc_en": "The final league boss. The Phoenix rises with colossal stats and FIRE attacks.",
        "reward_money": 5000,
        "reward_items": {"trophée de boss": 1},
        "achievement": "boss_league_5",
        "image_path": os.path.join(COG_DIR, "../../assets/bosses/stage_5.png")
    }
]

class BossLeagueView(discord.ui.View):
    def __init__(self, cog, ctx, stage, lang):
        super().__init__(timeout=120.0)
        self.cog = cog
        self.ctx = ctx
        self.stage = stage
        self.lang = lang
        self.fight_button.label = t("boss_league.fight_button", lang)

    @discord.ui.button(style=discord.ButtonStyle.danger, emoji="⚔️")
    async def fight_button(self, interaction: discord.Interaction, button: discord.ui.Button):
        if interaction.user.id != self.ctx.author.id:
            return await interaction.response.send_message("Ce n'est pas ton combat !", ephemeral=True)
        
        # Disable button
        for child in self.children:
            child.disabled = True
        await interaction.response.edit_message(view=self)
        
        # Start fight
        await self.cog.run_boss_fight(self.ctx, self.stage, self.lang)
        self.stop()

class BossLeague(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.locks = {} # server_id -> asyncio.Lock()

    @commands.group(name='league', aliases=['boss', 'boss_league'], invoke_without_command=True)
    async def league(self, ctx):
        """Affiche votre progression dans la Ligue des Boss."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = ctx.author.id
        stage = get_user_stage(user_id)

        if stage >= 5:
            embed = discord.Embed(
                title=t("boss_league.title", lang),
                description=t("boss_league.champion", lang),
                color=discord.Color.gold()
            )
            file = discord.File(CHAMPION_IMAGE_PATH, filename="champion.png")
            embed.set_thumbnail(url="attachment://champion.png")
            return await ctx.send(embed=embed, file=file)

        boss = BOSS_LEAGUE[stage]
        boss_name = boss["name_en"] if lang == 'en' else boss["name"]
        boss_desc = boss["desc_en"] if lang == 'en' else boss["desc"]
        boss_emoji = PETS_DB.get(boss["species"], {}).get("emoji", "🐾")

        # Format stats
        stats_txt = t("boss_league.stats_label", lang,
                      level=boss["level"],
                      species=boss["species"],
                      hp=boss["hp"],
                      atk=boss["atk"],
                      defense=boss["defense"],
                      speed=boss["speed"])

        # Format rewards
        rewards_txt = f"💵 **${boss['reward_money']}**\n"
        for item, qty in boss["reward_items"].items():
            rewards_txt += f"📦 {item.capitalize()} x{qty}\n"

        embed = discord.Embed(
            title=t("boss_league.title", lang),
            color=discord.Color.dark_red()
        )
        embed.add_field(name=t("boss_league.stage_info", lang, stage=stage + 1, name=boss_name),
                        value=f"*{boss_desc}*", inline=False)
        embed.add_field(name="Characteristics", value=f"{boss_emoji} {stats_txt}", inline=True)
        embed.add_field(name=t("boss_league.rewards_title", lang), value=rewards_txt, inline=True)
        embed.set_footer(text=t("boss_league.challenge_footer", lang))

        file = discord.File(boss["image_path"], filename=f"boss_{stage + 1}.png")
        embed.set_thumbnail(url=f"attachment://boss_{stage + 1}.png")

        view = BossLeagueView(self, ctx, stage, lang)
        await ctx.send(embed=embed, view=view, file=file)

    @league.command(name='fight')
    async def league_fight(self, ctx):
        """Combattre le boss de votre niveau actuel dans la Ligue."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = ctx.author.id
        stage = get_user_stage(user_id)

        if stage >= 5:
            return await ctx.send(t("boss_league.already_champion", lang))

        await self.run_boss_fight(ctx, stage, lang)

    async def run_boss_fight(self, ctx, stage, lang):
        user_id = ctx.author.id
        server_id = ctx.guild.id if ctx.guild else None

        # Lock server interaction during the fight to prevent concurrency issues
        if server_id not in self.locks:
            self.locks[server_id] = asyncio.Lock()

        async with self.locks[server_id]:
            # Load user pet
            user_pet = get_active_pet(user_id, server_id)
            if not user_pet:
                return await ctx.send(t("boss_league.no_pet", lang))
            if not user_pet.is_alive:
                return await ctx.send(t("boss_league.pet_ko", lang, name=user_pet.nickname))

            boss_config = BOSS_LEAGUE[stage]
            boss_name = boss_config["name_en"] if lang == 'en' else boss_config["name"]
            boss_emoji = PETS_DB.get(boss_config["species"], {}).get("emoji", "🐾")

            # Check inventory limit before starting the fight
            from src.database.housing import is_inventory_full
            full, current, limit = is_inventory_full(user_id)
            total_items_to_add = sum(boss_config["reward_items"].values())
            if current + total_items_to_add > limit:
                # Get translated warning message
                from src.utils.i18n import t
                warning_msg = t("housing.inv_full_warning", lang, current=current, limit=limit)
                return await ctx.send(warning_msg)

            # Create boss pet object
            boss_pet = Pet(
                pet_type=boss_config["species"],
                nickname=boss_name,
                level=boss_config["level"],
                max_hp=boss_config["hp"],
                hp=boss_config["hp"],
                atk=boss_config["atk"],
                defense=boss_config["defense"],
                speed=boss_config["speed"]
            )

            # Send challenge start message
            intro_msg = t("boss_league.fight_intro", lang,
                          pet_emoji=user_pet.emoji,
                          pet_name=user_pet.nickname,
                          boss_name=boss_name,
                          boss_emoji=boss_emoji)
            
            embed = discord.Embed(title=t("boss_league.title", lang), color=discord.Color.dark_red())
            
            def update_embed():
                embed.clear_fields()
                embed.add_field(
                    name=f"{user_pet.emoji} {user_pet.nickname} (Niv {user_pet.level})",
                    value=f"PV : {generate_hp_bar(user_pet.hp, user_pet.max_hp)}\n`{int(user_pet.hp)} / {user_pet.max_hp}`",
                    inline=True
                )
                embed.add_field(name="VS", value="⚡", inline=True)
                embed.add_field(
                    name=f"{boss_pet.emoji} {boss_pet.nickname} (Niv {boss_pet.level})",
                    value=f"PV : {generate_hp_bar(boss_pet.hp, boss_pet.max_hp)}\n`{int(boss_pet.hp)} / {boss_pet.max_hp}`",
                    inline=True
                )

            update_embed()
            embed.description = intro_msg
            file = discord.File(boss_config["image_path"], filename=f"boss_{stage + 1}.png")
            embed.set_thumbnail(url=f"attachment://boss_{stage + 1}.png")
            msg = await ctx.send(embed=embed, file=file)
            await asyncio.sleep(2)

            # Run battle simulation
            await simulate_battle(
                user_pet, boss_pet, msg, embed, update_embed,
                sleep_time=0.5, send_messages=True, log_size=10, lang=lang
            )

            # Save player's pet HP
            update_pet(user_pet)

            if user_pet.is_alive and not boss_pet.is_alive:
                # Victory! Increment stage
                new_stage = stage + 1
                set_user_stage(user_id, new_stage)

                # Distribute rewards
                update_balance(user_id, boss_config["reward_money"])
                for item, qty in boss_config["reward_items"].items():
                    add_item_to_inventory(user_id, item, qty)

                # Check and unlock achievements
                new_achievements = check_and_unlock_achievements(user_id)

                victory_embed = discord.Embed(
                    title=t("boss_league.victory", lang, boss_name=boss_name),
                    description=t("boss_league.champion", lang) if new_stage >= 5 else f"Vous passez à l'étape {new_stage + 1} !",
                    color=discord.Color.green()
                )
                
                # Format rewards message
                rewards_txt = f"💵 **+${boss_config['reward_money']}**\n"
                for item, qty in boss_config["reward_items"].items():
                    rewards_txt += f"📦 **{item.capitalize()}** x{qty}\n"
                
                victory_embed.add_field(name=t("boss_league.rewards_title", lang), value=rewards_txt, inline=False)

                # Show achievements unlocked
                if new_achievements:
                    ach_txt = ""
                    for ach in new_achievements:
                        ach_txt += f"{ach.emoji} **{ach.name(lang)}** (+{ach.glory} Gloire)\n"
                    victory_embed.add_field(name="🎖️ Succès Déverrouillés", value=ach_txt, inline=False)

                await ctx.send(embed=victory_embed)

            else:
                # Defeat!
                defeat_embed = discord.Embed(
                    title=t("boss_league.defeat", lang, pet_name=user_pet.nickname, boss_name=boss_name),
                    description="Entraînez votre familier et réessayez !",
                    color=discord.Color.red()
                )
                await ctx.send(embed=defeat_embed)

async def setup(bot):
    await bot.add_cog(BossLeague(bot))
