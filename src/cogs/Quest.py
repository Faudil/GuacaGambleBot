import discord
from discord.ext import commands
from discord.ui import View, Button
from typing import List, Optional

from src.models.Quest import QuestRegistry, QuestStepType, QuestType
from src.database.quest import get_user_quests, get_quest_progress, start_quest, is_quest_completed
from src.database.settings import get_language
from src.utils.QuestManager import QuestManager
from src.utils.i18n import t

class QuestDialogueView(View):
    def __init__(self, ctx, quest_id: str, lang: str):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.quest_id = quest_id
        self.lang = lang
        
        self.add_item(Button(
            label=t("quests.continue_label", lang),
            style=discord.ButtonStyle.primary,
            custom_id="quest_continue"
        ))

    async def interaction_check(self, interaction: discord.Interaction) -> bool:
        if interaction.user != self.ctx.author:
            return False
        return True

    @discord.ui.button(label="placeholder", style=discord.ButtonStyle.primary, custom_id="quest_continue_btn")
    async def continue_callback(self, interaction: discord.Interaction, button: Button):
        pass

    async def on_timeout(self):
        self.clear_items()

class QuestChoiceView(View):
    def __init__(self, ctx, quest_id: str, choices: List[dict], lang: str):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.quest_id = quest_id
        self.lang = lang
        
        for choice in choices:
            btn = Button(
                label=t(f"{choice['text_ref']}_label", lang, default=choice['id']),
                style=discord.ButtonStyle.secondary,
                custom_id=f"choice_{choice['id']}"
            )
            btn.callback = self.make_callback(choice['id'])
            self.add_item(btn)

    def make_callback(self, choice_id: str):
        async def callback(interaction: discord.Interaction):
            if interaction.user != self.ctx.author: return
            QuestManager.advance_step(interaction.user.id, self.quest_id, choice_id=choice_id)
            await self.show_next_step(interaction)
        return callback

    async def show_next_step(self, interaction: discord.Interaction):
        cog = self.ctx.bot.get_cog("Quest")
        if cog:
            await cog.show_quest_status(interaction, self.quest_id, edit=True)

class QuestRequirementView(View):
    def __init__(self, ctx, quest_id: str, step, lang: str):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.quest_id = quest_id
        self.step = step
        self.lang = lang
        
        btn_label = t("quests.req_button", lang)
        self.add_item(Button(
            label=btn_label if not btn_label.startswith("quests.") else "Fulfill Requirement",
            style=discord.ButtonStyle.success,
            custom_id="quest_req_btn"
        ))

    async def interaction_check(self, interaction: discord.Interaction) -> bool:
        if isinstance(self.ctx, commands.Context):
            return interaction.user == self.ctx.author
        return interaction.user == self.ctx.user

    @discord.ui.button(label="Fulfill Requirement", style=discord.ButtonStyle.success, custom_id="quest_req_callback")
    async def req_callback(self, interaction: discord.Interaction, button: Button):
        from src.database.balance import get_balance, update_balance
        from src.database.item import has_item, remove_item_from_inventory
        
        user_id = interaction.user.id
        reqs = self.step.extra.get('requirements', {})
        consume = self.step.extra.get('consume_reqs', True)
        
        # Check money
        req_money = reqs.get('money', 0)
        if req_money > 0:
            if get_balance(user_id) < req_money:
                msg = t("quests.req_missing", self.lang)
                return await interaction.response.send_message(msg if not msg.startswith("quests.") else "❌ Missing requirements!", ephemeral=True)
                
        # Check items
        req_items = reqs.get('items', {})
        for item_name, qty in req_items.items():
            if not has_item(user_id, item_name, qty):
                msg = t("quests.req_missing", self.lang)
                return await interaction.response.send_message(msg if not msg.startswith("quests.") else "❌ Missing requirements!", ephemeral=True)
                
        # Fulfill
        if consume:
            if req_money > 0:
                update_balance(user_id, -req_money)
            for item_name, qty in req_items.items():
                remove_item_from_inventory(user_id, item_name, qty)
                
        msg = t("quests.req_done", self.lang)
        await interaction.response.send_message(msg if not msg.startswith("quests.") else "✅ Done!", ephemeral=True)
        QuestManager.advance_step(user_id, self.quest_id)
        
        cog = self.ctx.bot.get_cog("Quest")
        if cog:
            # Re-render next step using original context message but edit mode on the view's message
            # The context is complex because we just sent an ephemeral response. 
            # We can't edit the original message from the ephemeral interaction response edit_message.
            # But we can use the interaction.message.
            ctx_mock = interaction.message
            ctx_mock.author = interaction.user
            await cog.show_quest_status(ctx_mock, self.quest_id, edit=True)

class QuestBossView(View):
    def __init__(self, ctx, quest_id: str, step, lang: str):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.quest_id = quest_id
        self.step = step
        self.lang = lang
        
        btn_label = t("quests.boss_fight_btn", lang)
        self.add_item(Button(
            label=btn_label if not btn_label.startswith("quests.") else "Fight Boss",
            style=discord.ButtonStyle.danger,
            custom_id="quest_boss_btn"
        ))

    async def interaction_check(self, interaction: discord.Interaction) -> bool:
        if isinstance(self.ctx, commands.Context):
            return interaction.user == self.ctx.author
        return interaction.user == self.ctx.user

    @discord.ui.button(label="Fight Boss", style=discord.ButtonStyle.danger, custom_id="quest_boss_callback")
    async def boss_callback(self, interaction: discord.Interaction, button: Button):
        from src.database.pets import get_active_pet, update_pet
        from src.models.Pet import Pet
        from src.utils.battle import simulate_battle
        import asyncio
        
        user_id = interaction.user.id
        pet = get_active_pet(user_id)
        if not pet:
            msg = t("quests.boss_no_pet", self.lang)
            return await interaction.response.send_message(msg if not msg.startswith("quests.") else "❌ No active pet!", ephemeral=True)
        if not pet.is_alive:
            msg = t("quests.boss_ko", self.lang)
            return await interaction.response.send_message(msg if not msg.startswith("quests.") else "❌ Pet is KO!", ephemeral=True)
            
        boss_stats = self.step.extra.get('boss_stats', {})
        enemy = Pet(
            pet_type=boss_stats.get('name', 'Boss'),
            nickname=boss_stats.get('name', 'Boss'),
            level=boss_stats.get('level', 10),
            max_hp=boss_stats.get('hp', 100),
            hp=boss_stats.get('hp', 100),
            atk=boss_stats.get('atk', 15),
            defense=boss_stats.get('def', 10),
            speed=boss_stats.get('spd', 10),
            dge=boss_stats.get('dge', 5),
            acc=boss_stats.get('acc', 10),
            crit_c=boss_stats.get('crit_c', 5),
            crit_d=boss_stats.get('crit_d', 1.5)
        )
        enemy._wild_emoji = boss_stats.get('emoji', '👹')

        await interaction.response.edit_message(view=None)
        
        embed = discord.Embed(title=f"⚔️ {enemy.nickname}", color=discord.Color.red())
        
        def update_embed():
            from src.utils.embed_utils import generate_hp_bar
            embed.clear_fields()
            embed.add_field(
                name=f"{pet.emoji} {pet.nickname} (Niv {pet.level})",
                value=f"PV : {generate_hp_bar(pet.hp, pet.max_hp)}\n`{int(pet.hp)} / {pet.max_hp}`",
                inline=True
            )
            embed.add_field(name="VS", value="⚡", inline=True)
            embed.add_field(
                name=f"{enemy._wild_emoji} {enemy.nickname} (Niv {enemy.level})",
                value=f"PV : {generate_hp_bar(enemy.hp, enemy.max_hp)}\n`{int(enemy.hp)} / {enemy.max_hp}`",
                inline=True
            )

        update_embed()
        embed.description = "Le combat commence !"
        await interaction.message.edit(embed=embed)
        
        await asyncio.sleep(2)
        
        await simulate_battle(
            pet, enemy, interaction.message, embed, update_embed,
            sleep_time=1.5, send_messages=True, log_size=5,
            journal_title="Journal de Combat",
            lang=self.lang
        )
        
        if pet.is_alive and not enemy.is_alive:
            msg = t("quests.boss_win", self.lang)
            embed.set_footer(text=msg if not msg.startswith("quests.") else "🏆 You won!")
            embed.color = discord.Color.gold()
            await interaction.message.edit(embed=embed)
            
            QuestManager.advance_step(user_id, self.quest_id)
            await asyncio.sleep(3)
            cog = self.ctx.bot.get_cog("Quest")
            if cog:
                ctx_mock = interaction.message
                ctx_mock.author = interaction.user
                await cog.show_quest_status(ctx_mock, self.quest_id, edit=True)
        else:
            msg = t("quests.boss_lose_taunt", self.lang)
            embed.set_footer(text=msg if not msg.startswith("quests.") else "💀 The Boss Laughs!")
            embed.color = discord.Color.red()
            await interaction.message.edit(embed=embed)
            
        update_pet(pet)


class Quest(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    async def show_quest_status(self, interaction_or_ctx, quest_id: str, edit=False):
        user_id = interaction_or_ctx.author.id if isinstance(interaction_or_ctx, commands.Context) else interaction_or_ctx.user.id
        lang = get_language(interaction_or_ctx.guild.id if interaction_or_ctx.guild else None)
        
        quest = QuestRegistry.get_quest(quest_id)
        progress = get_quest_progress(user_id, quest_id)
        
        if not progress or not quest:
            return

        step_idx = progress['step_index']
        if step_idx >= len(quest.steps):
            embed = discord.Embed(
                title=quest.get_title(lang),
                description="✨ " + t("quests.completed", lang, default="Quête terminée !"),
                color=discord.Color.gold()
            )
            if edit:
                await interaction_or_ctx.response.edit_message(embed=embed, view=None)
            else:
                await interaction_or_ctx.send(embed=embed)
            return

        step = quest.steps[step_idx]
        
        embed = discord.Embed(
            title=f"{quest.get_title(lang)}",
            description=step.get_text(lang),
            color=discord.Color.blue()
        )
        embed.set_footer(text=t("quests.step_progress", lang, current=step_idx+1, total=len(quest.steps)))

        view = None
        if step.step_type == QuestStepType.DIALOGUE:
            view = View()
            btn = Button(label=t("quests.continue_label", lang), style=discord.ButtonStyle.primary)
            async def cont_callback(interaction: discord.Interaction):
                if interaction.user.id != user_id: return
                QuestManager.advance_step(user_id, quest_id)
                await self.show_quest_status(interaction, quest_id, edit=True)
            btn.callback = cont_callback
            view.add_item(btn)
            
        elif step.step_type == QuestStepType.CHOICE:
            view = QuestChoiceView(interaction_or_ctx if isinstance(interaction_or_ctx, commands.Context) else None, quest_id, step.extra.get('choices', []), lang)
            if isinstance(interaction_or_ctx, discord.Interaction):
                view.ctx = await self.bot.get_context(interaction_or_ctx.message)
            else:
                view.ctx = interaction_or_ctx

        elif step.step_type == QuestStepType.ACTIVITY:
            target = step.extra.get('target_count', 1)
            current = progress.get('progress_value', 0)
            activity_name = t(f"activities.{step.extra.get('target_stat')}", lang, default=step.extra.get('target_stat'))
            embed.add_field(
                name="📊 " + t("quests.progress", lang, default="Progression"),
                value=t("quests.step_activity_progress", lang, current=current, target=target, activity=activity_name)
            )

        elif step.step_type == QuestStepType.REQUIREMENT:
            from src.database.balance import get_balance
            from src.database.item import get_all_user_inventory
            user_inv = get_all_user_inventory(user_id)
            inv_dict = {i['name']: i['quantity'] for i in user_inv}
            
            reqs = step.extra.get('requirements', {})
            req_money = reqs.get('money', 0)
            status_lines = []
            
            # Show money
            if req_money > 0:
                current_money = get_balance(user_id)
                msg = t("quests.req_money_status", lang, current=current_money, required=req_money)
                if msg.startswith("quests."): msg = f"💰 Money: {current_money} / {req_money}"
                status_lines.append(msg)
                
            # Show items
            req_items = reqs.get('items', {})
            for item_name, qty in req_items.items():
                current_qty = inv_dict.get(item_name, 0)
                from src.utils.i18n import get_item_name
                loc_item_name = get_item_name(item_name, lang)
                msg = t("quests.req_item_status", lang, item=loc_item_name, current=current_qty, required=qty)
                if msg.startswith("quests."): msg = f"🎒 {loc_item_name}: {current_qty} / {qty}"
                status_lines.append(msg)

            embed.add_field(
                name="📋 " + t("quests.progress", lang, default="Progression"),
                value="\n".join(status_lines)
            )
            
            view = QuestRequirementView(interaction_or_ctx if isinstance(interaction_or_ctx, commands.Context) else None, quest_id, step, lang)
            if isinstance(interaction_or_ctx, discord.Interaction):
                view.ctx = await self.bot.get_context(interaction_or_ctx.message)
            else:
                view.ctx = interaction_or_ctx

        elif step.step_type == QuestStepType.BOSS_BATTLE:
            boss_stats = step.extra.get('boss_stats', {})
            enemy_name = boss_stats.get('name', 'Boss')
            enemy_emoji = boss_stats.get('emoji', '👹')
            enemy_lvl = boss_stats.get('level', 10)
            
            embed.add_field(
                name="⚔️ " + t("quests.progress", lang, default="Progression"),
                value=f"**Boss :** {enemy_emoji} {enemy_name} (Lvl {enemy_lvl})"
            )
            
            view = QuestBossView(interaction_or_ctx if isinstance(interaction_or_ctx, commands.Context) else None, quest_id, step, lang)
            if isinstance(interaction_or_ctx, discord.Interaction):
                view.ctx = await self.bot.get_context(interaction_or_ctx.message)
            else:
                view.ctx = interaction_or_ctx

        if edit:

            await interaction_or_ctx.response.edit_message(embed=embed, view=view)
        else:
            await interaction_or_ctx.send(embed=embed, view=view)

    @commands.command(name='quest', aliases=['q'])
    async def quest(self, ctx, quest_id: Optional[str] = None):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        active = get_user_quests(ctx.author.id, status='ACTIVE')
        
        if quest_id:
            return await self.show_quest_status(ctx, quest_id)

        if not active:
            return await ctx.send(t("quests.no_active", lang))

        embed = discord.Embed(title=t("quests.title", lang), color=discord.Color.green())
        desc = t("quests.active_list", lang) + "\n\n"
        for q_data in active:
            quest = QuestRegistry.get_quest(q_data['quest_id'])
            if quest:
                desc += f"🔹 **{quest.get_title(lang)}** (`!q {quest.id}`)\n"
        
        embed.description = desc
        await ctx.send(embed=embed)

    @commands.command(name='starttutorial', hidden=True)
    async def starttutorial(self, ctx):
        start_quest(ctx.author.id, "tutorial")
        await ctx.send("🚀 Tutoriel lancé ! Tape `!q tutorial` pour commencer.")

async def setup(bot):
    await bot.add_cog(Quest(bot))
