import discord
from discord.ext import commands, tasks

from src.command_decorators import daily_limit
from src.database.balance import get_balance, update_balance
from src.database.lotto import try_daily_lotto_bonus, get_lotto_state, increment_lotto_jackpot, reset_lotto
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks

from src.globals import CHANNEL_ID


class Lotto(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot
        self.ticket_price = 20
        self.daily_increase = 300
        self.daily_pot_increase.start()

    def cog_unload(self):
            self.daily_pot_increase.cancel()

    @tasks.loop(hours=1)
    async def daily_pot_increase(self):
        bonus_applied = try_daily_lotto_bonus(self.daily_increase)
        if bonus_applied:
            state = get_lotto_state()
            new_jackpot = state['jackpot']
            for guild in self.bot.guilds:
                channel = guild.get_channel(CHANNEL_ID)
                if channel:
                    embed = discord.Embed(title="📈 LOTO : Hausse Quotidienne", color=discord.Color.gold())
                    embed.description = (
                        f"🌞 Un nouveau jour se lève !\n"
                        f"La banque ajoute **${self.daily_increase}** à la cagnotte.\n\n"
                        f"💰 **NOUVEAU JACKPOT : ${new_jackpot}**\n"
                        f"Pour tenter sa chance: `!lotto <nombre entre 1 et 100>` (le prix du billet est de {self.ticket_price})"
                    )
                    await channel.send(embed=embed)


    @commands.command(name='lotto', aliases=['loto'])
    @daily_limit("loto", 3)
    async def play_lotto(self, ctx, number: int):
        """Acheter un ticket de Loto."""
        user = ctx.author
        if not (1 <= number <= 100):
            return await ctx.send("❌ Choisis un nombre entre 1 et 100.")
        if get_balance(user.id) < self.ticket_price:
            return await ctx.send(f"❌ Le ticket coûte ${self.ticket_price}. T'es fauché.")
        update_balance(user.id, -self.ticket_price)
        increment_stat(user.id, "lotto_participations")
        state = get_lotto_state()
        winning_number = state['winning_number']
        current_jackpot = state['jackpot']
        added_value = int(self.ticket_price)
        increment_lotto_jackpot(added_value)
        if number == winning_number:
            increment_stat(user.id, "lotto_won")
            update_balance(user.id, current_jackpot)
            new_target, new_pot = reset_lotto()
            embed = discord.Embed(title="🎰 JACKPOT !!! 🎰", color=discord.Color.gold())
            embed.description = (
                f"🎉 **INCROYABLE !** {user.mention} a trouvé le numéro **{number}** !\n\n"
                f"💰 Tu remportes la cagnotte de **${current_jackpot}** !\n"
                f"Le loto est réinitialisé. Nouveau jackpot : ${new_pot}."
            )
            embed.set_image(url="https://media.giphy.com/media/26tOZ42Mg6pbTUPvy/giphy.gif")
            await ctx.send(embed=embed)
            await ctx.guild.get_channel(CHANNEL_ID).send(
                f"🚨 **ALERTE LOTO** : {user.display_name} vient de gagner le JACKPOT de ${current_jackpot} !")
            
            unlocks = check_and_unlock_achievements(user.id)
            if unlocks:
                await ctx.send(embed=format_achievements_unlocks(unlocks))
            return
        else:
            embed = discord.Embed(title="🎫 Ticket validé", color=discord.Color.blue())
            embed.description = (
                f"Tu as joué le **{number}**.\n"
                f"Dommage, ce n'est pas le bon numéro.\n\n"
                f"📈 Le Jackpot augmente de **${added_value}** !\n"
                f"💰 **Cagnotte actuelle : ${current_jackpot + added_value}**"
            )
            embed.set_footer(text="Reviens demain pour un nouvel essai !")
            await ctx.send(embed=embed)
            
            unlocks = check_and_unlock_achievements(user.id)
            if unlocks:
                await ctx.send(embed=format_achievements_unlocks(unlocks))
            return

    @daily_pot_increase.before_loop
    async def before_pot_increase(self):
        await self.bot.wait_until_ready()

    @commands.command(name='jackpot')
    async def show_jackpot(self, ctx):
        """Voir la cagnotte actuelle du Loto."""
        state = get_lotto_state()
        embed = discord.Embed(title="💰 Cagnotte du Loto", color=discord.Color.green())
        embed.description = f"Le jackpot est actuellement de **${state['jackpot']}** !\nCoût du ticket : ${self.ticket_price}"
        embed.set_footer(text="Tape !loto <nombre> pour jouer")
        await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Lotto(bot))