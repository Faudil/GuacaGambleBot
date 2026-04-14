import discord
from discord.ext import commands
import random
from discord.ui import Button, View

from src.database.balance import update_balance, get_balance
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.settings import get_language
from src.utils.i18n import t


class RouletteGameView(View):
    def __init__(self, players, entry_fee, lang):
        super().__init__(timeout=300)
        self.players = players
        self.entry_fee = entry_fee
        self.lang = lang
        self.alive_players = players.copy()
        self.cylinder = [False] * 6
        self.cylinder[random.randint(0, 5)] = True
        self.turn_index = 0
        self.pot = len(players) * entry_fee

    def get_current_player(self):
        return self.alive_players[self.turn_index % len(self.alive_players)]

    @discord.ui.button(label="trigger", style=discord.ButtonStyle.danger)
    async def trigger(self, interaction: discord.Interaction, button: Button):
        shooter = self.get_current_player()
        if interaction.user != shooter:
            return await interaction.response.send_message(t("roulette.not_your_turn", self.lang), ephemeral=True)
        bullet = self.cylinder.pop(0)
        if bullet:
            await interaction.response.send_message(t("roulette.bang_msg", self.lang, user=shooter.mention))
            self.stop()
            increment_stat(shooter.id, "roulette_lost")
            increment_stat(shooter.id, "roulette_spent", self.entry_fee)
            increment_stat(shooter.id, "roulette_money_lost", self.entry_fee)
            
            survivors = [p for p in self.alive_players if p != shooter]
            share = self.pot // len(survivors)
            text = t("roulette.survivors_win", self.lang, user=shooter.display_name, fee=self.entry_fee, count=len(survivors), share=share)
            embed = discord.Embed(title=t("roulette.finish_title", self.lang), description=text, color=discord.Color.red())
            await interaction.channel.send(embed=embed)
            for s in survivors:
                update_balance(s.id, share)
                increment_stat(s.id, "roulette_won")
                increment_stat(s.id, "roulette_spent", self.entry_fee)
                
                net_win = share - self.entry_fee
                if net_win > 0:
                    increment_stat(s.id, "roulette_money_won", net_win)
                    
                unlocks = check_and_unlock_achievements(s.id)
                if unlocks:
                    await interaction.channel.send(content=s.mention, embed=format_achievements_unlocks(unlocks, self.lang))
            return
        else:
            await interaction.response.send_message(t("roulette.clic_msg", self.lang, user=shooter.display_name))
            self.turn_index += 1
            next_player = self.get_current_player()

            embed = discord.Embed(description=t("roulette.next_turn", self.lang, user=next_player.mention),
                                  color=discord.Color.dark_grey())
            return await interaction.channel.send(embed=embed)


class JoinView(View):
    def __init__(self, ctx, amount, lang):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.amount = amount
        self.lang = lang
        self.players = []

    @discord.ui.button(label="join", style=discord.ButtonStyle.blurple)
    async def join(self, interaction, button):
        if interaction.user in self.players:
            return await interaction.response.send_message(t("roulette.already_joined", self.lang), ephemeral=True)

        if get_balance(interaction.user.id) < self.amount:
            return await interaction.response.send_message(t("roulette.no_money", self.lang), ephemeral=True)

        update_balance(interaction.user.id, -self.amount)
        self.players.append(interaction.user)
        return await interaction.response.send_message(t("roulette.joined_msg", self.lang, user=interaction.user.display_name),
                                                ephemeral=False)

    @discord.ui.button(label="start", style=discord.ButtonStyle.green)
    async def start(self, interaction, button):
        if interaction.user != self.ctx.author:
            return await interaction.response.send_message(t("roulette.only_leader", self.lang),
                                                           ephemeral=True)
        if len(self.players) < 2:
            return await interaction.response.send_message(t("roulette.min_players", self.lang), ephemeral=True)
        self.stop()
        game_view = RouletteGameView(self.players, self.amount, self.lang)
        # Update button label
        for item in game_view.children:
            if isinstance(item, Button) and item.label == "trigger":
                item.label = t("roulette.trigger_label", self.lang)

        first_player = game_view.get_current_player()
        return await interaction.channel.send(
            t("roulette.start_announcement", self.lang, user=first_player.mention),
            view=game_view
        )


class Roulette(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='roulette', aliases=['rr'])
    async def roulette(self, ctx, amount: int):
        """Roulette Russe. 1 chance sur 6 de mourir (et perdre la mise)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if amount <= 0:
            return await ctx.send(t("blackjack.invalid_bet", lang))
        
        view = JoinView(ctx, amount, lang)
        # Update button labels
        for item in view.children:
            if isinstance(item, Button):
                if item.label == "join": item.label = t("roulette.join_label", lang)
                if item.label == "start": item.label = t("roulette.start_label", lang)

        return await ctx.send(
            t("roulette.open_msg", lang, amount=amount, user=ctx.author.mention),
            view=view)


async def setup(bot):
    await bot.add_cog(Roulette(bot))