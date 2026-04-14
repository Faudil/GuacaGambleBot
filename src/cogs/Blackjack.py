import discord
from discord.ext import commands
import random
from discord.ui import Button, View

from src.database.balance import update_balance, get_balance
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.settings import get_language
from src.utils.i18n import t


def create_deck():
    suits = ["♠️", "♥️", "♦️", "♣️"]
    ranks = ["2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"]
    values = {r: 10 if r in "JQK" else (11 if r == "A" else int(r)) for r in ranks}
    deck = [(s, r, values[r]) for s in suits for r in ranks]
    random.shuffle(deck)
    return deck


def calculate_score(hand):
    score = sum(card[2] for card in hand)
    aces = sum(1 for card in hand if card[1] == "A")
    while score > 21 and aces:
        score -= 10
        aces -= 1
    return score


def format_hand(hand):
    return " ".join([f"[{c[1]}{c[0]}]" for c in hand])


class BlackjackView(View):
    def __init__(self, bot, ctx, p1, p2, amount, deck, lang):
        super().__init__(timeout=120)
        self.embed = None
        self.bot = bot
        self.ctx = ctx
        self.p1 = p1
        self.p2 = p2
        self.amount = amount
        self.deck = deck
        self.lang = lang

        self.turn = p1
        self.hands = {p1: [deck.pop(), deck.pop()], p2: [deck.pop(), deck.pop()]}
        self.scores = {p1: calculate_score(self.hands[p1]), p2: calculate_score(self.hands[p2])}
        self.finished = {p1: False, p2: False}
        self.update_embed()

    def update_embed(self):
        """Met à jour l'affichage du jeu."""
        score_p1 = calculate_score(self.hands[self.p1])
        score_p2 = calculate_score(self.hands[self.p2])
        
        status_p1 = t("blackjack.player_turn", self.lang, user="") if self.turn == self.p1 else ""
        
        status_p2 = ""
        if self.turn == self.p1 and not self.finished[self.p1]:
            status_p2 = t("blackjack.player_waiting", self.lang, user="")
        elif self.turn == self.p2:
            status_p2 = t("blackjack.player_turn", self.lang, user="")

        desc = (
            t("blackjack.pot", self.lang, amount=self.amount * 2) + "\n\n"
            f"👤 **{self.p1.display_name}** {status_p1}\n"
            f"{t('blackjack.hand', self.lang, hand=format_hand(self.hands[self.p1]), score=score_p1)}\n\n"
            f"👤 **{self.p2.display_name}** {status_p2}\n"
            f"{t('blackjack.hand', self.lang, hand=format_hand(self.hands[self.p2]), score=score_p2)}"
        )

        color = discord.Color.blue() if self.turn == self.p1 else discord.Color.purple()
        self.embed = discord.Embed(title=t("blackjack.title", self.lang), description=desc, color=color)
        self.embed.set_footer(text=t("blackjack.footer", self.lang, user=self.turn.display_name))

    async def check_game_over(self, interaction):
        s1 = calculate_score(self.hands[self.p1])
        s2 = calculate_score(self.hands[self.p2])
        winner = None
        reason = ""

        if s1 > 21:
            winner = self.p2
            reason = t("blackjack.bust", self.lang, user=self.p1.display_name)
        elif s2 > 21:
            winner = self.p1
            reason = t("blackjack.bust", self.lang, user=self.p2.display_name)
        elif self.finished[self.p1] and self.finished[self.p2]:
            if s1 > s2:
                winner = self.p1
                reason = t("blackjack.beat", self.lang, s1=s1, s2=s2)
            elif s2 > s1:
                winner = self.p2
                reason = t("blackjack.beat", self.lang, s1=s2, s2=s1)
            else:
                winner = "DRAW"
                reason = t("blackjack.draw", self.lang)
        if winner:
            await self.end_game(interaction, winner, reason)
            return True
        return False

    async def end_game(self, interaction, winner, reason):
        """Gère la distribution des gains et la fin de l'interaction."""
        self.stop()
        self.clear_items()
        if winner == "DRAW":
            update_balance(self.p1.id, self.amount)
            update_balance(self.p2.id, self.amount)
            result_text = t("blackjack.draw_msg", self.lang, reason=reason)
            color = discord.Color.light_grey()
            
            from src.database.npc import add_reputation
            rep1 = add_reputation(self.p1.id, "gamblebot", 5)
            rep2 = add_reputation(self.p2.id, "gamblebot", 5)
            if rep1 > 0 or rep2 > 0:
                result_text += "\n🤖 + Rép. GambleBot"
        else:
            update_balance(winner.id, self.amount * 2)
            loser = self.p1 if winner == self.p2 else self.p2
            increment_stat(winner.id, "blackjack_won")
            increment_stat(loser.id, "blackjack_lost")
            
            increment_stat(winner.id, "blackjack_spent", self.amount)
            increment_stat(winner.id, "blackjack_money_won", self.amount)
            
            increment_stat(loser.id, "blackjack_spent", self.amount)
            increment_stat(loser.id, "blackjack_money_lost", self.amount)
            
            result_text = t("blackjack.win_msg", self.lang, user=winner.display_name, reason=reason, amount=self.amount * 2)
            color = discord.Color.gold()
            
            from src.database.npc import add_reputation
            rep_w = add_reputation(winner.id, "gamblebot", 10)
            rep_l = add_reputation(loser.id, "gamblebot", 5)
            if rep_w > 0 or rep_l > 0:
                result_text += "\n🤖 + Rép. GambleBot"

        final_embed = self.embed
        final_embed.color = color
        final_embed.description += f"\n\n{t('blackjack.game_over', self.lang)}\n{result_text}"
        await interaction.response.edit_message(embed=final_embed, view=None)
        
        if winner != "DRAW":
            w_unlocks = check_and_unlock_achievements(winner.id)
            l_unlocks = check_and_unlock_achievements(loser.id)
            if w_unlocks:
                await interaction.channel.send(embed=format_achievements_unlocks(w_unlocks, self.lang), content=winner.mention)
            if l_unlocks:
                await interaction.channel.send(embed=format_achievements_unlocks(l_unlocks, self.lang), content=loser.mention)

    @discord.ui.button(label="hit", style=discord.ButtonStyle.success)
    async def hit(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.turn:
            return await interaction.response.send_message(t("blackjack.not_your_turn", self.lang), ephemeral=True)
        card = self.deck.pop()
        self.hands[self.turn].append(card)
        score = calculate_score(self.hands[self.turn])
        if score > 21:
            self.finished[self.turn] = True
            self.update_embed()
            return await self.check_game_over(interaction)
        else:
            self.update_embed()
            return await interaction.response.edit_message(embed=self.embed, view=self)

    @discord.ui.button(label="stand", style=discord.ButtonStyle.danger)
    async def stand(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.turn:
            return await interaction.response.send_message(t("blackjack.not_your_turn", self.lang), ephemeral=True)

        self.finished[self.turn] = True

        if self.turn == self.p1:
            self.turn = self.p2
            self.update_embed()
            return await interaction.response.edit_message(embed=self.embed, view=self)
        else:
            self.update_embed()
            return await self.check_game_over(interaction)


class BlackjackPvP(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.challenges = {}

    @commands.command(name='bjduel', aliases=['bjpvp', 'blackjack', 'bj'])
    async def bjduel(self, ctx, opponent: discord.Member, amount: int):
        """Le 21. Affronte un autre joueur."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        challenger = ctx.author
        if opponent.bot or opponent.id == challenger.id:
            return await ctx.send(t("blackjack.invalid_opponent", lang))
        if amount <= 0:
            return await ctx.send(t("blackjack.invalid_bet", lang))
        bal_p1 = get_balance(challenger.id)
        bal_p2 = get_balance(opponent.id)

        if bal_p1 < amount:
            return await ctx.send(t("blackjack.no_money_self", lang, balance=bal_p1))
        if bal_p2 < amount:
            return await ctx.send(t("blackjack.no_money_opponent", lang, user=opponent.display_name, balance=bal_p2))
        
        view = View()
        accept_btn = Button(label=t("blackjack.accept_label", lang), style=discord.ButtonStyle.green)

        async def accept_callback(interaction):
            if interaction.user != opponent:
                return await interaction.response.send_message(t("item_manager.not_for_you", lang), ephemeral=True)
            if get_balance(challenger.id) < amount or get_balance(opponent.id) < amount:
                return await interaction.response.send_message(t("blackjack.no_money_problem", lang))
            update_balance(challenger.id, -amount)
            update_balance(opponent.id, -amount)
            deck = create_deck()
            game_view = BlackjackView(self.bot, ctx, challenger, opponent, amount, deck, lang)
            # Override button labels after initialization if they were not passed correctly
            for item in game_view.children:
                if isinstance(item, Button):
                    if item.label == "hit": item.label = t("blackjack.hit_label", lang)
                    if item.label == "stand": item.label = t("blackjack.stand_label", lang)

            return await interaction.response.edit_message(content=None, embed=game_view.embed, view=game_view)
        
        accept_btn.callback = accept_callback
        view.add_item(accept_btn)
        return await ctx.send(t("blackjack.challenge_msg", lang, challenger=challenger.mention, opponent=opponent.mention, amount=amount), view=view)


async def setup(bot):
    await bot.add_cog(BlackjackPvP(bot))