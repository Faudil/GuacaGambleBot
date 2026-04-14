import discord
from discord.ext import commands, tasks

from src.command_decorators import daily_limit
from src.database.balance import get_balance, update_balance
from src.database.lotto import try_daily_lotto_bonus, get_lotto_state, increment_lotto_jackpot, reset_lotto
from src.database.achievement import increment_stat, check_and_unlock_achievements, format_achievements_unlocks
from src.database.settings import get_announcement_channel, get_language
from src.utils.i18n import t


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
        for guild in self.bot.guilds:
            lang = get_language(guild.id)
            bonus_applied = try_daily_lotto_bonus(guild.id, self.daily_increase)
            if bonus_applied:
                state = get_lotto_state(guild.id)
                new_jackpot = state['jackpot']
                
                channel_id = get_announcement_channel(guild.id)
                if channel_id == 0: continue
                channel = guild.get_channel(channel_id) if channel_id else guild.system_channel
                if not channel and guild.text_channels: channel = guild.text_channels[0]
                
                if channel:
                    embed = discord.Embed(title=t("lotto.daily_hausse_title", lang), color=discord.Color.gold())
                    embed.description = t("lotto.daily_hausse_desc", lang, amount=self.daily_increase, jackpot=new_jackpot, price=self.ticket_price)
                    await channel.send(embed=embed)


    @commands.command(name='lotto', aliases=['loto'])
    @daily_limit("loto", 3)
    async def play_lotto(self, ctx, number: int):
        """Acheter un ticket de Loto."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user = ctx.author
        if not (1 <= number <= 100):
            return await ctx.send(t("lotto.invalid_number", lang))
        if get_balance(user.id) < self.ticket_price:
            return await ctx.send(t("lotto.no_money", lang, price=self.ticket_price))
            
        update_balance(user.id, -self.ticket_price)
        increment_stat(user.id, "lotto_participations")
        state = get_lotto_state(ctx.guild.id)
        winning_number = state['winning_number']
        current_jackpot = state['jackpot']
        added_value = int(self.ticket_price)
        increment_lotto_jackpot(ctx.guild.id, added_value)
        
        if number == winning_number:
            increment_stat(user.id, "lotto_won")
            update_balance(user.id, current_jackpot)
            new_target, new_pot = reset_lotto(ctx.guild.id)
            embed = discord.Embed(title=t("lotto.jackpot_title", lang), color=discord.Color.gold())
            embed.description = t("lotto.jackpot_win_desc", lang, user=user.mention, number=number, jackpot=current_jackpot, new_pot=new_pot)
            embed.set_image(url="https://media.giphy.com/media/26tOZ42Mg6pbTUPvy/giphy.gif")
            await ctx.send(embed=embed)
            
            channel_id = get_announcement_channel(ctx.guild.id)
            if channel_id != 0:
                channel = ctx.guild.get_channel(channel_id) if channel_id else ctx.guild.system_channel
                if not channel and ctx.guild.text_channels: channel = ctx.guild.text_channels[0]
                if channel:
                    await channel.send(t("lotto.announcement_alert", lang, user=user.display_name, jackpot=current_jackpot))
            
            unlocks = check_and_unlock_achievements(user.id)
            if unlocks:
                await ctx.send(embed=format_achievements_unlocks(unlocks, lang))
            return
        else:
            embed = discord.Embed(title=t("lotto.ticket_valid_title", lang), color=discord.Color.blue())
            embed.description = t("lotto.ticket_valid_desc", lang, number=number, added=added_value, total=current_jackpot + added_value)
            embed.set_footer(text=t("lotto.ticket_valid_footer", lang))
            await ctx.send(embed=embed)
            
            unlocks = check_and_unlock_achievements(user.id)
            if unlocks:
                await ctx.send(embed=format_achievements_unlocks(unlocks, lang))
            return

    @daily_pot_increase.before_loop
    async def before_pot_increase(self):
        await self.bot.wait_until_ready()

    @commands.command(name='jackpot')
    async def show_jackpot(self, ctx):
        """Voir la cagnotte actuelle du Loto."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        state = get_lotto_state(ctx.guild.id)
        embed = discord.Embed(title=t("lotto.show_jackpot_title", lang), color=discord.Color.green())
        embed.description = t("lotto.show_jackpot_desc", lang, jackpot=state['jackpot'], price=self.ticket_price)
        embed.set_footer(text=t("lotto.show_jackpot_footer", lang))
        await ctx.send(embed=embed)


async def setup(bot):
    await bot.add_cog(Lotto(bot))