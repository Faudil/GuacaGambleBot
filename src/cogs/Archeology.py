import asyncio
import random

import discord
from discord.ext import commands
from discord.ui import View, Button

from src.command_decorators import daily_limit
from src.database.balance import get_balance, update_balance
from src.database.item import add_item_to_inventory, has_item, remove_item_from_inventory
from src.items.ArcheologyLoot import (
    BoneDust, BrokenFossil, CommonFossil, RareFossil, EpicFossil, LegendaryFragment, PureDNA
)
from src.models.Pet import Pet
from src.database.pets import insert_new_pet
from src.database.settings import get_language
from src.utils.i18n import t, get_item_name, get_pet_name


class ExtractionView(View):
    def __init__(self, ctx, permit_type, lang):
        super().__init__(timeout=120)
        self.ctx = ctx
        self.permit_type = permit_type
        self.lang = lang
        self.depth = 50
        self.integrity = 100
        self.actions = 5
        self.finished = False

    def generate_embed(self):
        embed = discord.Embed(title=t("archeology.mini_game_title", self.lang), color=discord.Color.dark_gold())

        depth_percentage = max(0.0, self.depth / 50.0)
        blocks_empty = int((1.0 - depth_percentage) * 5)
        blocks_full = 5 - blocks_empty
        depth_bar = ("🟫" * blocks_full) + ("⬛" * blocks_empty)

        int_percentage = max(0.0, self.integrity / 100.0)
        int_blocks = round(int_percentage * 5)
        int_bar = ("❤️" * int_blocks) + ("💔" * (5 - int_blocks))

        embed.add_field(name=t("archeology.depth_remaining", self.lang), value=f"{depth_bar} {self.depth} cm", inline=False)
        embed.add_field(name=t("archeology.integrity", self.lang), value=f"{int_bar} {self.integrity}%", inline=False)
        embed.add_field(name=t("archeology.actions_remaining", self.lang), value=f"**{self.actions}**", inline=False)
        
        permit_name = t("archeology.safe_site", self.lang) if self.permit_type == "safe" else t("archeology.fault_site", self.lang)
        embed.set_footer(text=t("archeology.permit_footer", self.lang, name=permit_name))
        return embed

    async def update_message(self, interaction: discord.Interaction):
        if self.finished:
            for child in self.children:
                child.disabled = True
            await interaction.response.edit_message(embed=self.generate_embed(), view=self)
            await self.resolve_game(interaction.message)
        else:
            await interaction.response.edit_message(embed=self.generate_embed(), view=self)

    async def apply_action(self, interaction: discord.Interaction, depth_removed: int, risk_chance: int, int_loss: int):
        if interaction.user != self.ctx.author:
            return

        self.actions -= 1
        self.depth = max(0, self.depth - depth_removed)

        final_risk = risk_chance
        if self.permit_type == "safe" and risk_chance > 0:
            final_risk = risk_chance // 2

        roll = random.randint(1, 100)
        if roll <= final_risk:
            self.integrity = max(0, self.integrity - int_loss)

        if self.depth <= 0 or self.integrity <= 0 or self.actions <= 0:
            self.finished = True

        await self.update_message(interaction)

    @discord.ui.button(label="dynamite_placeholder", style=discord.ButtonStyle.danger)
    async def btn_dynamite(self, interaction: discord.Interaction, button: Button):
        await self.apply_action(interaction, depth_removed=20, risk_chance=50, int_loss=30)

    @discord.ui.button(label="hammer_placeholder", style=discord.ButtonStyle.primary)
    async def btn_hammer(self, interaction: discord.Interaction, button: Button):
        await self.apply_action(interaction, depth_removed=10, risk_chance=15, int_loss=10)

    @discord.ui.button(label="brush_placeholder", style=discord.ButtonStyle.success)
    async def btn_brush(self, interaction: discord.Interaction, button: Button):
        await self.apply_action(interaction, depth_removed=2, risk_chance=0, int_loss=0)

    async def resolve_game(self, message: discord.Message):
        if self.integrity <= 0:
            item = BoneDust()
            msg = t("archeology.disaster_msg", self.lang)
        elif self.depth > 0 and self.actions <= 0:
            item = BoneDust()
            msg = t("archeology.timeout_msg", self.lang)
        else:
            if self.integrity < 50:
                item = BrokenFossil()
                msg = t("archeology.damaged_msg", self.lang)
            elif self.integrity == 100:
                item = PureDNA()
                msg = t("archeology.perfect_msg", self.lang, item=get_item_name(item.name, self.lang))
            else:
                if self.permit_type == "safe":
                    roll = random.random()
                    if roll < 0.60:
                        item = CommonFossil()
                    elif roll < 0.90:
                        item = RareFossil()
                    else:
                        item = EpicFossil()
                    msg = t("archeology.success_msg", self.lang, item=get_item_name(item.name, self.lang), integrity=self.integrity)
                else:
                    item = LegendaryFragment()
                    msg = t("archeology.legendary_msg", self.lang, item=get_item_name(item.name, self.lang), integrity=self.integrity)

        add_item_to_inventory(self.ctx.author.id, item.name)

        embed = self.generate_embed()
        embed.description = f"{msg}\n{t('archeology.received', self.lang, item=get_item_name(item.name, self.lang))}"
        await message.edit(embed=embed, view=None)


class PermitView(View):
    def __init__(self, ctx, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.lang = lang
        self.chosen_permit = None

    @discord.ui.button(label="safe_permit_placeholder", style=discord.ButtonStyle.success)
    async def btn_safe(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author:
            return
        self.chosen_permit = "safe"
        await self.start_extraction(interaction)

    @discord.ui.button(label="fault_permit_placeholder", style=discord.ButtonStyle.danger)
    async def btn_faille(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author:
            return

        bal = get_balance(self.ctx.author.id)
        if bal < 200:
            await interaction.response.send_message(t("archeology.no_money_permit", self.lang), ephemeral=True)
            return

        update_balance(self.ctx.author.id, -200)
        self.chosen_permit = "faille"
        await self.start_extraction(interaction)

    async def start_extraction(self, interaction: discord.Interaction):
        self.clear_items()
        extraction_view = ExtractionView(self.ctx, self.chosen_permit, self.lang)
        for item in extraction_view.children:
            if isinstance(item, Button):
                if item.label == "dynamite_placeholder": item.label = t("archeology.dynamite_label", self.lang)
                if item.label == "hammer_placeholder": item.label = t("archeology.hammer_label", self.lang)
                if item.label == "brush_placeholder": item.label = t("archeology.brush_label", self.lang)

        await interaction.response.edit_message(content=t("archeology.searching_site", self.lang), embed=None, view=None)
        await asyncio.sleep(1)
        await interaction.message.edit(content=None, embed=extraction_view.generate_embed(), view=extraction_view)
        self.stop()


class Archeology(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.group(name="dig", aliases=["archeology", "archeologie"], invoke_without_command=True)
    @daily_limit("dig", 5)
    async def dig(self, ctx):
        """Launcher une expédition de fouilles archéologiques selon 2 niveaux de permis."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        
        from src.database.housing import is_inventory_full
        full, current, limit = is_inventory_full(ctx.author.id)
        if full:
            return await ctx.send(t("housing.inv_full_warning", lang, current=current, limit=limit))
            
        embed = discord.Embed(
            title=t("archeology.bureau_title", lang),
            description=(
                t("archeology.bureau_desc", lang) +
                t("archeology.safe_desc", lang) +
                t("archeology.fault_desc", lang)
            ),
            color=discord.Color.blue()
        )
        view = PermitView(ctx, lang)
        for item in view.children:
            if isinstance(item, Button):
                 if item.label == "safe_permit_placeholder": item.label = t("archeology.safe_permit_label", lang)
                 if item.label == "fault_permit_placeholder": item.label = t("archeology.fault_permit_label", lang)

        await ctx.send(embed=embed, view=view)

    @commands.command(name="reanimate", aliases=["rea"])
    async def reanimate(self, ctx, *, rarity: str = None):
        """Réanimmer un animal à l'aide de 5 pièces fossilisées de même rareté."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if not rarity:
            await ctx.send(t("archeology.reanimate_no_rarity", lang))
            return

        pools = {
            "commun": {"class": CommonFossil, "pets": ["Escargot", "Souris", "Cochon", "Grenouille", "Mouton"]},
            "common": {"class": CommonFossil, "pets": ["Escargot", "Souris", "Cochon", "Grenouille", "Mouton"]},
            "rare": {"class": RareFossil, "pets": ["Chien", "Chat", "Cheval", "Renard", "Singe", "Ours"]},
            "épique": {"class": EpicFossil, "pets": ["Chameau", "Panda", "Tigre", "Pieuvre"]},
            "epique": {"class": EpicFossil, "pets": ["Chameau", "Panda", "Tigre", "Pieuvre"]},
            "epic": {"class": EpicFossil, "pets": ["Chameau", "Panda", "Tigre", "Pieuvre"]},
            "légendaire": {"class": LegendaryFragment, "pets": ["Dragon", "Tyrannosaure", "Diplodocus", "Mamouth"]},
            "legendaire": {"class": LegendaryFragment, "pets": ["Dragon", "Tyrannosaure", "Diplodocus", "Mamouth"]},
            "legendary": {"class": LegendaryFragment, "pets": ["Dragon", "Tyrannosaure", "Diplodocus", "Mamouth"]},
            "adn pur": {"class": PureDNA, "pets": ["Mégalodon", "Kraken", "Licorne", "Phoenix", "Cerbère"]},
            "pure dna": {"class": PureDNA, "pets": ["Mégalodon", "Kraken", "Licorne", "Phoenix", "Cerbère"]}
        }
        
        rarity_key = rarity.lower().strip()
        if rarity_key not in pools:
            await ctx.send(t("archeology.reanimate_invalid_rarity", lang))
            return

        pool = pools[rarity_key]
        c_class = pool["class"]()

        if not has_item(ctx.author.id, c_class.name, 5):
            await ctx.send(t("archeology.reanimate_no_parts", lang, item=get_item_name(c_class.name, lang)))
            return

        remove_item_from_inventory(ctx.author.id, c_class.name, 5)

        pet_name_key = random.choice(pool["pets"])
        pet = Pet.create_new(ctx.author.id, pet_name_key)
        insert_new_pet(pet)

        await ctx.send(t("archeology.reanimate_success", lang, name=get_pet_name(pet_name_key, lang), emoji=pet.emoji))


async def setup(bot):
    await bot.add_cog(Archeology(bot))
