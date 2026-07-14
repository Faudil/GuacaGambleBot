import discord
from discord.ext import commands
from discord.ui import View, Button, Select

from src.models.NPC import NPCRegistry
from src.database.npc import get_reputation, add_reputation
from src.database.settings import get_language
from src.database.quest import start_quest, is_quest_completed, get_user_quests
from src.database.item import get_all_user_inventory, remove_item_from_inventory
from src.models.Quest import QuestRegistry
from src.utils.i18n import t

def make_progress_bar(current, max_val):
    pct = min(1.0, max(0.0, current / max_val)) if max_val > 0 else 1.0
    filled = int(pct * 10)
    empty = 10 - filled
    return "█" * filled + "░" * empty + f" {int(pct * 100)}%"

class NPCTalkTopicView(View):
    def __init__(self, ctx, npc, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.npc = npc
        self.lang = lang

        options = [
            discord.SelectOption(label=t("npcs.topic_bio", lang), value="bio", emoji="💬"),
            discord.SelectOption(label=t("npcs.topic_role", lang), value="role", emoji="⚙️"),
            discord.SelectOption(label=t("npcs.topic_advice", lang), value="advice", emoji="💡"),
            discord.SelectOption(label=t("npcs.topic_chat", lang), value="chat", emoji="🍃")
        ]
        select = Select(placeholder=t("npcs.talk_menu_placeholder", lang), options=options)
        select.callback = self.select_callback
        self.add_item(select)

        # Add a back button
        back_btn = Button(label="↩️", style=discord.ButtonStyle.secondary)
        back_btn.callback = self.back_callback
        self.add_item(back_btn)

    async def select_callback(self, interaction: discord.Interaction):
        if interaction.user != self.ctx.author: return
        topic = interaction.data['values'][0]
        desc = t(f"npcs.{self.npc.id}.{topic}", self.lang)
        
        embed = discord.Embed(
            title=f"{self.npc.emoji} {self.npc.name} - {t(f'npcs.topic_{topic}', self.lang)}",
            description=desc,
            color=self.npc.color
        )
        await interaction.response.edit_message(embed=embed, view=self)

    async def back_callback(self, interaction: discord.Interaction):
        if interaction.user != self.ctx.author: return
        view = NPCInteractionView(self.ctx, self.npc, self.lang)
        embed = view.get_main_embed()
        await interaction.response.edit_message(embed=embed, view=view)

class NPCGiftConfirmView(View):
    def __init__(self, ctx, npc, item, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.npc = npc
        self.item = item
        self.lang = lang

        # Add buttons for different quantities
        qty = item['quantity']
        
        self.add_item(Button(label=f"Gift 1", style=discord.ButtonStyle.primary, custom_id="gift_1"))
        if qty >= 5:
            self.add_item(Button(label=f"Gift 5", style=discord.ButtonStyle.primary, custom_id="gift_5"))
        self.add_item(Button(label=f"Gift All ({qty})", style=discord.ButtonStyle.success, custom_id="gift_all"))
        
        back_btn = Button(label="↩️", style=discord.ButtonStyle.secondary, custom_id="back")
        self.add_item(back_btn)

        for btn in self.children:
            btn.callback = self.button_callback

    async def button_callback(self, interaction: discord.Interaction):
        if interaction.user != self.ctx.author: return
        
        custom_id = interaction.data['custom_id']
        if custom_id == "back":
            # Go back to item selection
            inv = get_all_user_inventory(interaction.user.id)
            view = NPCGiftSelectionView(self.ctx, self.npc, inv, self.lang)
            hint = t(f"npcs.{self.npc.id}.hint", self.lang)
            content = t("npcs.gift_prompt", self.lang, hint=hint)
            return await interaction.response.edit_message(content=content, embed=None, view=view)

        # Calculate gift size
        if custom_id == "gift_1":
            qty_to_gift = 1
        elif custom_id == "gift_5":
            qty_to_gift = 5
        else: # gift_all
            qty_to_gift = self.item['quantity']

        # Perform the gifting
        remove_item_from_inventory(interaction.user.id, self.item['name'], qty_to_gift)
        
        points = self.npc.on_gift(interaction.user.id, self.item['name'], qty_to_gift)
        actual_added = add_reputation(interaction.user.id, self.npc.id, points)
        
        desc = t("npcs.gift_success", self.lang, qty=qty_to_gift, item=self.item['name'], name=self.npc.name, points=actual_added)
        if actual_added < points:
            desc += t("npcs.gift_cap_warning", self.lang, lost=(points - actual_added))
            
        embed = discord.Embed(
            title=f"🎁 Cadeau pour {self.npc.name}",
            description=desc,
            color=self.npc.color
        )

        # Show back button to main NPC screen
        back_view = View()
        back_btn = Button(label="↩️ Menu", style=discord.ButtonStyle.secondary)
        async def go_back_to_menu(it: discord.Interaction):
            if it.user != self.ctx.author: return
            m_view = NPCInteractionView(self.ctx, self.npc, self.lang)
            m_embed = m_view.get_main_embed()
            await it.response.edit_message(content=None, embed=m_embed, view=m_view)
        back_btn.callback = go_back_to_menu
        back_view.add_item(back_btn)

        await interaction.response.edit_message(content=None, embed=embed, view=back_view)

class NPCGiftSelectionView(View):
    def __init__(self, ctx, npc, user_inventory, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.npc = npc
        self.lang = lang
        self.inventory = user_inventory
        
        options = []
        for item in user_inventory:
            mult = npc.gift_preferences.get(item['name'].lower(), 1.0)
            if mult == 2.0:
                affinity_tag = " ❤️ (Loved)" if lang == "en" else " ❤️ (Adoré)"
            elif mult == 1.5:
                affinity_tag = " 👍 (Liked)" if lang == "en" else " 👍 (Aimé)"
            elif mult == 0.5:
                affinity_tag = " 👎 (Disliked)" if lang == "en" else " 👎 (Détesté)"
            else:
                affinity_tag = " 😐"
                
            options.append(discord.SelectOption(
                label=f"{item['quantity']}x {item['name']}{affinity_tag}",
                value=str(item['id']),
                description=item['description'][:100] if item.get('description') else None
            ))

        if not options:
            self.add_item(Button(label=t("npcs.empty_inventory", lang), disabled=True))
        else:
            select = Select(placeholder=t("npcs.gift_prompt", lang, hint=""), options=options[:25])
            select.callback = self.select_callback
            self.add_item(select)

        # Back button to main menu
        back_btn = Button(label="↩️", style=discord.ButtonStyle.secondary)
        back_btn.callback = self.back_callback
        self.add_item(back_btn)

    async def select_callback(self, interaction: discord.Interaction):
        if interaction.user != self.ctx.author: return
        
        item_id = int(interaction.data['values'][0])
        item = next((i for i in self.inventory if i['id'] == item_id), None)
        
        if not item: return

        # Open quantity selection view
        view = NPCGiftConfirmView(self.ctx, self.npc, item, self.lang)
        embed = discord.Embed(
            title=f"🎁 Offrir {item['name']}",
            description=f"Combien de **{item['name']}** veux-tu offrir à {self.npc.name} ?\n(Tu en possèdes : **{item['quantity']}**)",
            color=self.npc.color
        )
        await interaction.response.edit_message(content=None, embed=embed, view=view)

    async def back_callback(self, interaction: discord.Interaction):
        if interaction.user != self.ctx.author: return
        view = NPCInteractionView(self.ctx, self.npc, self.lang)
        embed = view.get_main_embed()
        await interaction.response.edit_message(content=None, embed=embed, view=view)

class NPCInteractionView(View):
    def __init__(self, ctx, npc, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.npc = npc
        self.lang = lang

    def get_main_embed(self) -> discord.Embed:
        rep_data = get_reputation(self.ctx.author.id, self.npc.id)
        lvl = rep_data["level"]
        points = rep_data["reputation"]
        next_lvl = 100 * lvl
        rank_name = self.npc.get_rank_name(lvl, self.lang)
        bar = make_progress_bar(points, next_lvl)

        msg = self.npc.get_greeting(self.ctx.author.id, self.lang)
        embed = discord.Embed(
            title=f"{self.npc.emoji} {self.npc.name}",
            description=f"*{msg}*\n\n**Affinité** : {rank_name} (Lvl {lvl})\n{bar} ({points}/{next_lvl})",
            color=self.npc.color
        )
        return embed

    @discord.ui.button(label="Parler", style=discord.ButtonStyle.primary)
    async def talk_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        view = NPCTalkTopicView(self.ctx, self.npc, self.lang)
        embed = self.get_main_embed()
        await interaction.response.edit_message(embed=embed, view=view)

    @discord.ui.button(label="Offrir un cadeau", style=discord.ButtonStyle.secondary)
    async def gift_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        
        inv = get_all_user_inventory(interaction.user.id)
        if not inv:
            return await interaction.response.send_message(t("npcs.empty_inventory", self.lang), ephemeral=True)
            
        view = NPCGiftSelectionView(self.ctx, self.npc, inv, self.lang)
        hint = t(f"npcs.{self.npc.id}.hint", self.lang)
        content = t("npcs.gift_prompt", self.lang, hint=hint)
        await interaction.response.edit_message(content=content, embed=None, view=view)

    @discord.ui.button(label="Quêtes", style=discord.ButtonStyle.success)
    async def quests_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        
        all_quests = QuestRegistry.get_all_quests()
        available = [q for q in all_quests if q.npc_id == self.npc.id and q.is_available(interaction.user.id)]
        
        if not available:
            return await interaction.response.send_message(f"{self.npc.name} n'a aucune quête pour toi en ce moment.", ephemeral=True)
            
        embed = discord.Embed(title=f"📜 Quêtes de {self.npc.name}", color=self.npc.color)
        desc = ""
        for q in available:
            desc += f"🔹 **{q.get_title(self.lang)}**\n"
            
        embed.description = desc
        
        # Add back button to menu
        back_view = View()
        back_btn = Button(label="↩️ Menu", style=discord.ButtonStyle.secondary)
        async def go_back_to_menu(it: discord.Interaction):
            if it.user != self.ctx.author: return
            m_view = NPCInteractionView(self.ctx, self.npc, self.lang)
            m_embed = m_view.get_main_embed()
            await it.response.edit_message(embed=m_embed, view=m_view)
        back_btn.callback = go_back_to_menu
        back_view.add_item(back_btn)

        await interaction.response.edit_message(embed=embed, view=back_view)

    @discord.ui.button(label="Promouvoir", style=discord.ButtonStyle.danger)
    async def rankup_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        from src.database.npc import rank_up_npc
        rep_data = get_reputation(interaction.user.id, self.npc.id)
        if rep_data["reputation"] < 100 * rep_data["level"]:
            return await interaction.response.send_message(t("npcs.rankup_no_rep", self.lang), ephemeral=True)
            
        costs = self.npc.get_rank_up_cost(rep_data["level"])
        if not costs:
            return await interaction.response.send_message(t("npcs.rankup_max", self.lang), ephemeral=True)
            
        inv = get_all_user_inventory(interaction.user.id)
        for cost in costs:
            item_in_inv = next((i for i in inv if i['name'] == cost['name']), None)
            if not item_in_inv or item_in_inv['quantity'] < cost['quantity']:
                return await interaction.response.send_message(t("npcs.rankup_cost_missing", self.lang, item=cost['name'], qty=cost['quantity']), ephemeral=True)
                
        # Consume resources and rank up
        for cost in costs:
            remove_item_from_inventory(interaction.user.id, cost['name'], cost['quantity'])
            
        rank_up_npc(interaction.user.id, self.npc.id)
        new_rank = self.npc.get_rank_name(rep_data["level"] + 1, self.lang)
        
        embed = discord.Embed(
            title="🎉 Promotion !",
            description=t("npcs.rankup_success", self.lang, rank=new_rank, name=self.npc.name),
            color=0x00ff00
        )
        # Add back button to menu
        back_view = View()
        back_btn = Button(label="↩️ Menu", style=discord.ButtonStyle.secondary)
        async def go_back_to_menu(it: discord.Interaction):
            if it.user != self.ctx.author: return
            m_view = NPCInteractionView(self.ctx, self.npc, self.lang)
            m_embed = m_view.get_main_embed()
            await it.response.edit_message(embed=m_embed, view=m_view)
        back_btn.callback = go_back_to_menu
        back_view.add_item(back_btn)

        await interaction.response.edit_message(embed=embed, view=back_view)

    @discord.ui.button(label="Boutique", style=discord.ButtonStyle.success, row=1)
    async def shop_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author:
            return None
        from src.cogs.Shop import DailyShopView
        rep_data = get_reputation(interaction.user.id, self.npc.id)
        items = self.npc.get_shop_inventory(rep_data["level"])
        if not items:
            return await interaction.response.send_message(t("npcs.shop_empty", self.lang, name=self.npc.name), ephemeral=True)
            
        offers = [{'item': item, 'price': item.price, 'discounted': False} for item in items]
        view = DailyShopView(interaction.user, offers, self.lang)
        embed = discord.Embed(title=t("npcs.shop_title", self.lang, name=self.npc.name), color=self.npc.color)
        return await interaction.response.send_message(embed=embed, view=view, ephemeral=True)

class NPCs(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='npc', aliases=['npcs'])
    async def npc_list(self, ctx):
        """Affiche la liste des NPCs rencontrés."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        all_npcs = NPCRegistry.get_all_npcs()
        
        embed = discord.Embed(title=t("npcs.list_title", lang), color=discord.Color.blue())
        
        view = View()
        for npc in all_npcs:
            rep_data = get_reputation(ctx.author.id, npc.id)
            lvl = rep_data["level"]
            points = rep_data["reputation"]
            next_lvl = 100 * lvl
            rank_name = npc.get_rank_name(lvl, lang)
            bar = make_progress_bar(points, next_lvl)
            
            embed.add_field(
                name=f"{npc.emoji} {npc.name} ({rank_name})",
                value=f"Lvl {lvl} - {bar} ({points}/{next_lvl})",
                inline=False
            )
            
            async def cb_factory(n):
                async def cb(it: discord.Interaction):
                    if it.user.id != ctx.author.id:
                        return
                    iview = NPCInteractionView(ctx, n, lang)
                    m_embed = iview.get_main_embed()
                    await it.response.send_message(embed=m_embed, view=iview, ephemeral=True)
                return cb
            
            b = Button(label=t("npcs.talk_button", lang, name=npc.name), style=discord.ButtonStyle.secondary)
            b.callback = await cb_factory(npc)
            view.add_item(b)

        await ctx.send(embed=embed, view=view)

    @commands.command(name='talk', hidden=True)
    async def talk(self, ctx, npc_id: str):
        """Parler à un NPC spécifique."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        npc = NPCRegistry.get_npc(npc_id)
        if not npc:
            return await ctx.send("Personnage inconnu.")
            
        view = NPCInteractionView(ctx, npc, lang)
        embed = view.get_main_embed()
        await ctx.send(embed=embed, view=view)

async def setup(bot):
    await bot.add_cog(NPCs(bot))
