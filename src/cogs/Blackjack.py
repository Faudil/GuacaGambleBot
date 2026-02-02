import discord
from discord.ext import commands
import random
from discord.ui import Button, View

# Imports avec le chemin absolu correct
from src.data_handling import get_balance, update_balance


# --- LOGIQUE DES CARTES ---
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
    def __init__(self, bot, ctx, p1, p2, amount, deck):
        super().__init__(timeout=120)
        self.embed = None
        self.bot = bot
        self.ctx = ctx
        self.p1 = p1
        self.p2 = p2
        self.amount = amount
        self.deck = deck

        self.turn = p1
        self.hands = {p1: [deck.pop(), deck.pop()], p2: [deck.pop(), deck.pop()]}
        self.scores = {p1: calculate_score(self.hands[p1]), p2: calculate_score(self.hands[p2])}
        self.finished = {p1: False, p2: False}
        self.update_embed()

    def update_embed(self):
        """Met à jour l'affichage du jeu."""
        score_p1 = calculate_score(self.hands[self.p1])
        score_p2 = calculate_score(self.hands[self.p2])
        desc = (
            f"💰 **Pot : ${self.amount * 2}**\n\n"
            f"👤 **{self.p1.display_name}** {'(En cours...)' if self.turn == self.p1 else ''}\n"
            f"Main : {format_hand(self.hands[self.p1])} (**{score_p1}**)\n\n"
            f"👤 **{self.p2.display_name}** {'(En attente)' if self.turn == self.p1 and not self.finished[self.p1] else '(En cours...)' if self.turn == self.p2 else ''}\n"
            f"Main : {format_hand(self.hands[self.p2])} (**{score_p2}**)"
        )

        color = discord.Color.blue() if self.turn == self.p1 else discord.Color.purple()
        self.embed = discord.Embed(title="🃏 Blackjack Duel", description=desc, color=color)
        self.embed.set_footer(text=f"C'est au tour de {self.turn.display_name}")

    async def check_game_over(self, interaction):
        """Vérifie si le jeu est fini et détermine le gagnant."""
        s1 = calculate_score(self.hands[self.p1])
        s2 = calculate_score(self.hands[self.p2])
        winner = None
        reason = ""

        if s1 > 21:
            winner = self.p2
            reason = f"{self.p1.display_name} a sauté (Bust) !"
        elif s2 > 21:
            winner = self.p1
            reason = f"{self.p2.display_name} a sauté (Bust) !"
        elif self.finished[self.p1] and self.finished[self.p2]:
            if s1 > s2:
                winner = self.p1
                reason = f"{s1} bat {s2}"
            elif s2 > s1:
                winner = self.p2
                reason = f"{s2} bat {s1}"
            else:
                winner = "DRAW"
                reason = "Égalité parfaite !"
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
            result_text = f"🤝 **Égalité !** ({reason})\nVos mises ont été remboursées."
            color = discord.Color.light_grey()
        else:
            update_balance(winner.id, self.amount * 2)
            result_text = f"🎉 **{winner.display_name} remporte le duel !** ({reason})\nIl gagne **${self.amount * 2}**."
            color = discord.Color.gold()

        final_embed = self.embed
        final_embed.color = color
        final_embed.description += f"\n\n🛑 **FIN DU JEU**\n{result_text}"
        await interaction.response.edit_message(embed=final_embed, view=None)

    @discord.ui.button(label="Tirer (Hit)", style=discord.ButtonStyle.success)
    async def hit(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.turn:
            return await interaction.response.send_message("Ce n'est pas ton tour !", ephemeral=True)
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

    @discord.ui.button(label="Rester (Stand)", style=discord.ButtonStyle.danger)
    async def stand(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.turn:
            return await interaction.response.send_message("Ce n'est pas ton tour !", ephemeral=True)

        self.finished[self.turn] = True

        # Si c'était P1, on passe à P2
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

    @commands.command(name='bjduel', aliases=['bjpvp'])
    async def bjduel(self, ctx, opponent: discord.Member, amount: int):
        """Défie quelqu'un au Blackjack. Usage: !bjduel @Pseudo 100"""
        challenger = ctx.author
        if opponent.bot or opponent.id == challenger.id:
            return await ctx.send("❌ Adversaire invalide.")
        if amount <= 0:
            return await ctx.send("❌ Mise invalide.")
        bal_p1 = get_balance(challenger.id)
        bal_p2 = get_balance(opponent.id)

        if bal_p1 < amount:
            return await ctx.send(f"❌ Tu n'as pas assez d'argent (${bal_p1}).")
        if bal_p2 < amount:
            return await ctx.send(f"❌ {opponent.display_name} n'a pas assez d'argent (${bal_p2}).")
        view = View()
        accept_btn = Button(label="Accepter le Duel", style=discord.ButtonStyle.green)

        async def accept_callback(interaction):
            if interaction.user != opponent:
                return await interaction.response.send_message("Ce défi n'est pas pour toi.", ephemeral=True)
            if get_balance(challenger.id) < amount or get_balance(opponent.id) < amount:
                return await interaction.response.send_message("❌ Problème de fonds, duel annulé.")
            update_balance(challenger.id, -amount)
            update_balance(opponent.id, -amount)
            deck = create_deck()
            game_view = BlackjackView(self.bot, ctx, challenger, opponent, amount, deck)
            return await interaction.response.edit_message(content=None, embed=game_view.embed, view=game_view)
        accept_btn.callback = accept_callback
        view.add_item(accept_btn)
        return await ctx.send(f"🃏 **{challenger.mention}** défie **{opponent.mention}** au Blackjack pour **${amount}** !",
                       view=view)


async def setup(bot):
    await bot.add_cog(BlackjackPvP(bot))