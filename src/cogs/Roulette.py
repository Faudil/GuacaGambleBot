import discord
from discord.ext import commands
import random
from discord.ui import Button, View

from src.database.balance import update_balance, get_balance
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks


class RouletteGameView(View):
    def __init__(self, players, entry_fee):
        super().__init__(timeout=300)
        self.players = players
        self.entry_fee = entry_fee
        self.alive_players = players.copy()
        self.cylinder = [False] * 6
        self.cylinder[random.randint(0, 5)] = True
        self.turn_index = 0
        self.pot = len(players) * entry_fee

    def get_current_player(self):
        return self.alive_players[self.turn_index % len(self.alive_players)]

    @discord.ui.button(label="😱 Presser la détente", style=discord.ButtonStyle.danger)
    async def trigger(self, interaction: discord.Interaction, button: Button):
        shooter = self.get_current_player()
        if interaction.user != shooter:
            return await interaction.response.send_message("Ce n'est pas ton tour !", ephemeral=True)
        bullet = self.cylinder.pop(0)
        if bullet:
            await interaction.response.send_message(f"💥 **BANG !** {shooter.mention} est mort...")
            self.stop()
            increment_stat(shooter.id, "roulette_lost")
            increment_stat(shooter.id, "roulette_spent", self.entry_fee)
            increment_stat(shooter.id, "roulette_money_lost", self.entry_fee)
            
            survivors = [p for p in self.alive_players if p != shooter]
            share = self.pot // len(survivors)
            text = f"💀 **{shooter.display_name}** a perdu **${self.entry_fee}**.\n"
            text += f"💰 Les {len(survivors)} survivants gagnent chacun **${share}** !"
            embed = discord.Embed(title="🩸 FIN DE LA PARTIE", description=text, color=discord.Color.red())
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
                    await interaction.channel.send(content=s.mention, embed=format_achievements_unlocks(unlocks))
            return
        else:
            await interaction.response.send_message(f"😅 **CLIC !** {shooter.display_name} survit...")
            self.turn_index += 1
            next_player = self.get_current_player()

            embed = discord.Embed(description=f"C'est au tour de {next_player.mention} de prendre l'arme.",
                                  color=discord.Color.dark_grey())
            return await interaction.channel.send(embed=embed)


class JoinView(View):
    def __init__(self, ctx, amount):
        super().__init__(timeout=60)
        self.ctx = ctx
        self.amount = amount
        self.players = []

    @discord.ui.button(label="Rejoindre la partie", style=discord.ButtonStyle.blurple)
    async def join(self, interaction, button):
        if interaction.user in self.players:
            return await interaction.response.send_message("Tu es déjà inscrit.", ephemeral=True)

        if get_balance(interaction.user.id) < self.amount:
            return await interaction.response.send_message("Pas assez d'argent !", ephemeral=True)

        update_balance(interaction.user.id, -self.amount)
        self.players.append(interaction.user)
        return await interaction.response.send_message(f"✅ {interaction.user.display_name} a rejoint la table !",
                                                ephemeral=False)

    @discord.ui.button(label="Lancer (Leader)", style=discord.ButtonStyle.green)
    async def start(self, interaction, button):
        if interaction.user != self.ctx.author:
            return await interaction.response.send_message("Seul celui qui a créé la partie peut la lancer.",
                                                           ephemeral=True)
        if len(self.players) < 2:
            return await interaction.response.send_message("Il faut au moins 2 joueurs !", ephemeral=True)
        self.stop()
        game_view = RouletteGameView(self.players, self.amount)
        first_player = game_view.get_current_player()
        return await interaction.channel.send(
            f"🔫 **La partie commence !** Il y a 1 balle dans le barillet.\nC'est à {first_player.mention} de tirer.",
            view=game_view
        )


class Roulette(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='roulette', aliases=['rr'])
    async def roulette(self, ctx, amount: int):
        """Roulette Russe. 1 chance sur 6 de mourir (et perdre la mise)."""
        if amount <= 0:
            return await ctx.send("Mise invalide.")
        view = JoinView(ctx, amount)
        return await ctx.send(
            f"🔫 **Roulette Russe ouverte !** Mise : **${amount}**\nCliquez pour rejoindre. {ctx.author.mention}, lance quand tout le monde est prêt.",
            view=view)


async def setup(bot):
    await bot.add_cog(Roulette(bot))