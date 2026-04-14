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

class NPCGiftSelectionView(View):
    def __init__(self, ctx, npc, user_inventory, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.npc = npc
        self.lang = lang
        self.inventory = user_inventory
        
        options = []
        for item in user_inventory:
            options.append(discord.SelectOption(
                label=f"{item['quantity']}x {item['name']}",
                value=str(item['id']),
                description=item['description'][:100] if item.get('description') else None
            ))

        if not options:
            self.add_item(Button(label="Inventaire vide", disabled=True))
        else:
            select = Select(placeholder="Choisis un objet à offrir...", options=options[:25])
            select.callback = self.select_callback
            self.add_item(select)

    async def select_callback(self, interaction: discord.Interaction):
        if interaction.user != self.ctx.author: return
        
        item_id = int(interaction.data['values'][0])
        item = next((i for i in self.inventory if i['id'] == item_id), None)
        
        if not item: return

        remove_item_from_inventory(interaction.user.id, item['name'], 1)
        
        points = self.npc.on_gift(interaction.user.id, item['name'], 1)
        actual_added = add_reputation(interaction.user.id, self.npc.id, points)
        
        desc = f"Tu as offert **1x {item['name']}** à {self.npc.name} !\n📈 Points gagnés : **{actual_added}**"
        if actual_added < points:
            desc += f"\n*(La limite journalière est atteinte, {points - actual_added} points perdus)*"
            
        embed = discord.Embed(
            title=f"🎁 Cadeau pour {self.npc.name}",
            description=desc,
            color=self.npc.color
        )
        await interaction.response.edit_message(embed=embed, view=None)

class NPCInteractionView(View):
    def __init__(self, ctx, npc, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.npc = npc
        self.lang = lang

    @discord.ui.button(label="Parler", style=discord.ButtonStyle.primary)
    async def talk_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        msg = self.npc.get_greeting(interaction.user.id, self.lang)
        embed = discord.Embed(title=f"💬 {self.npc.name}", description=msg, color=self.npc.color)
        await interaction.response.edit_message(embed=embed)

    @discord.ui.button(label="Offrir un cadeau", style=discord.ButtonStyle.secondary)
    async def gift_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        
        inv = get_all_user_inventory(interaction.user.id)
        
        if not inv:
            return await interaction.response.send_message("Ton inventaire est vide !", ephemeral=True)
            
        view = NPCGiftSelectionView(self.ctx, self.npc, inv, self.lang)
        await interaction.response.edit_message(content="Que veux-tu offrir ?", view=view)

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
        await interaction.response.edit_message(embed=embed)

    @discord.ui.button(label="Promouvoir", style=discord.ButtonStyle.danger)
    async def rankup_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author: return
        from src.database.npc import rank_up_npc
        rep_data = get_reputation(interaction.user.id, self.npc.id)
        if rep_data["reputation"] < 100 * rep_data["level"]:
            return await interaction.response.send_message("Tu n'as pas assez de réputation pour passer au rang suivant !", ephemeral=True)
            
        costs = self.npc.get_rank_up_cost(rep_data["level"])
        if not costs:
            return await interaction.response.send_message("Tu as atteint le rang maximum !", ephemeral=True)
            
        inv = get_all_user_inventory(interaction.user.id)
        for cost in costs:
            item_in_inv = next((i for i in inv if i['name'] == cost['name']), None)
            if not item_in_inv or item_in_inv['quantity'] < cost['quantity']:
                return await interaction.response.send_message(f"Tu n'as pas assez de {cost['name']} ! Il t'en faut {cost['quantity']}.", ephemeral=True)
                
        # Consume resources and rank up
        for cost in costs:
            remove_item_from_inventory(interaction.user.id, cost['name'], cost['quantity'])
            
        rank_up_npc(interaction.user.id, self.npc.id)
        new_rank = self.npc.get_rank_name(rep_data["level"] + 1, self.lang)
        await interaction.response.send_message(f"🎉 Félicitations ! Tu as atteint le rang **{new_rank}** avec {self.npc.name} !", ephemeral=False)

    @discord.ui.button(label="Boutique", style=discord.ButtonStyle.success, row=1)
    async def shop_btn(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.ctx.author:
            return None
        from src.cogs.Shop import DailyShopView
        rep_data = get_reputation(interaction.user.id, self.npc.id)
        items = self.npc.get_shop_inventory(rep_data["level"])
        if not items:
            return await interaction.response.send_message(f"{self.npc.name} n'a rien à te vendre pour le moment.", ephemeral=True)
            
        offers = [{'item': item, 'price': item.price, 'discounted': False} for item in items]
        view = DailyShopView(interaction.user, offers, self.lang)
        embed = discord.Embed(title=f"🛍️ Boutique de {self.npc.name}", color=self.npc.color)
        return await interaction.response.send_message(embed=embed, view=view, ephemeral=True)

class NPCs(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='npc', aliases=['npcs'])
    async def npc_list(self, ctx):
        """Affiche la liste des NPCs rencontrés."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        all_npcs = NPCRegistry.get_all_npcs()
        
        embed = discord.Embed(title="👥 Personnages du Guacamolistan", color=discord.Color.blue())
        
        view = View()
        for npc in all_npcs:
            rep_data = get_reputation(ctx.author.id, npc.id)
            lvl = rep_data["level"]
            points = rep_data["reputation"]
            next_lvl = 100 * lvl
            rank_name = npc.get_rank_name(lvl, lang)
            
            embed.add_field(
                name=f"{npc.emoji} {npc.name}",
                value=f"Affinité: {rank_name} (Lvl {lvl}) - ({points}/{next_lvl})",
                inline=False
            )
            
            btn = Button(label=f"Parler à {npc.name}", custom_id=f"talk_{npc.id}", style=discord.ButtonStyle.secondary)
            # Use a helper to capture the current npc in the closure
            btn.callback = self.get_callback(npc, lang)
            view.add_item(btn)

        await ctx.send(embed=embed, view=view)

    def get_callback(self, npc, lang):
        async def callback(interaction: discord.Interaction):
            if interaction.user.id != interaction.user.id:
                return
            view = NPCInteractionView(self.bot, npc, lang)
            pass
        return None

    @commands.command(name='talk', hidden=True)
    async def talk(self, ctx, npc_id: str):
        """Parlet à un NPC spécifique."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        npc = NPCRegistry.get_npc(npc_id)
        if not npc:
            return await ctx.send("Personnage inconnu.")
            
        view = NPCInteractionView(ctx, npc, lang)
        msg = npc.get_greeting(ctx.author.id, lang)
        embed = discord.Embed(title=f"{npc.emoji} {npc.name}", description=msg, color=npc.color)
        await ctx.send(embed=embed, view=view)

async def setup(bot):
    cog = NPCs(bot)
    
    async def npc_list_cmd(self, ctx):
        lang = get_language(ctx.guild.id if ctx.guild else None)
        all_npcs = NPCRegistry.get_all_npcs()
        embed = discord.Embed(title="👥 Personnages", color=discord.Color.blue())
        view = View()
        for npc in all_npcs:
            rep_data = get_reputation(ctx.author.id, npc.id)
            lvl = rep_data["level"]
            points = rep_data["reputation"]
            next_lvl = 100 * lvl
            rank_name = npc.get_rank_name(lvl, lang)
            embed.add_field(name=f"{npc.emoji} {npc.name}", value=f"Affinité: {rank_name} (Lvl {lvl}) - ({points}/{next_lvl})", inline=False)
            
            async def cb_factory(n):
                async def cb(it: discord.Interaction):
                    if it.user.id != ctx.author.id:
                        return
                    iview = NPCInteractionView(ctx, n, lang)
                    msg = n.get_greeting(ctx.author.id, lang)
                    e = discord.Embed(title=f"{n.emoji} {n.name}", description=msg, color=n.color)
                    await it.response.send_message(embed=e, view=iview, ephemeral=True)
                return cb
            
            b = Button(label=f"Parler à {npc.name}", style=discord.ButtonStyle.secondary)
            b.callback = await cb_factory(npc)
            view.add_item(b)
        await ctx.send(embed=embed, view=view)
        
    cog.npc_list.callback = npc_list_cmd.__get__(cog, NPCs)
    await bot.add_cog(cog)
