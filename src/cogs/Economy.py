import random

import discord
from discord.ext import commands, tasks

from src.command_decorators import daily_limit
from src.database.achievement import check_and_unlock_achievements, format_achievements_unlocks
from src.database.balance import update_balance, get_balance
from src.database.bank import get_bank_data
from src.database.loan import get_total_debt, repay_debt_logic
from src.database.other import pay_random_broke_user
from src.database.settings import get_announcement_channel, get_language
from src.globals import DAILY_AMOUNT
from src.utils.i18n import t

from src.models.quests.DailyQuest import DailyQuest
from src.database.quest import start_quest


class Economy(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot
        self.random_welfare.start()

    def cog_unload(self):
        self.random_welfare.cancel()

    @tasks.loop(minutes=30)
    async def random_welfare(self):
        if random.random() < 0.7:
            amount = 150
            winner_id = pay_random_broke_user(amount, max_balance=200)
            if winner_id:
                for guild in self.bot.guilds:
                    member = guild.get_member(winner_id)
                    if member:
                        lang = get_language(guild.id)
                        channel_id = get_announcement_channel(guild.id)
                        if channel_id == 0: continue
                        channel = guild.get_channel(channel_id) if channel_id else guild.system_channel
                        if not channel and guild.text_channels:
                            channel = guild.text_channels[0]
                        if channel:
                            embed = discord.Embed(
                                title=t("economy.rsa_title", lang),
                                description=t("economy.rsa_desc", lang, user=member.mention, amount=amount),
                                color=discord.Color.green()
                            )
                            embed.set_thumbnail(url=member.display_avatar.url)
                            await channel.send(embed=embed)
                            break

    @random_welfare.before_loop
    async def before_welfare(self):
        await self.bot.wait_until_ready()

    @commands.command(name='balance', aliases=['bal'])
    async def balance(self, ctx, member: discord.Member = None):
        """Voir ton solde."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user = ctx.author if member is None else member
        wallet, bank = get_bank_data(user.id)
        interest = (bank // 100) * 10
        embed = discord.Embed(title=t("economy.balance_title", lang), color=discord.Color.blue())
        embed.add_field(name=t("economy.wallet", lang), value=f"${wallet}", inline=True)
        embed.add_field(name=t("economy.safe", lang), value=f"${bank} / 500", inline=True)
        embed.add_field(name=t("economy.daily_interest", lang), value=f"+${interest} / jour", inline=False)
        embed.set_footer(text=t("economy.balance_footer", lang))
        await ctx.send(embed=embed)

    @commands.command(name='daily')
    @daily_limit("daily", 1)
    async def daily(self, ctx):
        """Ton salaire journalier."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        user_id = str(ctx.author.id)
        _, bank_bal = get_bank_data(user_id)

        amount = DAILY_AMOUNT + (bank_bal // 100) * 10

        debt = get_total_debt(user_id)
        repay_cut = int(amount // 2)
        actual_repay = min(repay_cut, debt)

        player_gain = amount - actual_repay
        new_balance = update_balance(user_id, player_gain)
        paid, lenders = repay_debt_logic(user_id, actual_repay)

        embed = discord.Embed(title=t("economy.daily_title", lang), color=discord.Color.green())
        embed.add_field(name=t("economy.quantity", lang), value=f"+${amount}")
        if debt > 0:
            embed.add_field(name=t("economy.tax_repayment", lang), value=f"-${actual_repay}", inline=False)
            for lender, famount in lenders:
                embed.add_field(name=t("economy.repaid_lender", lang, lender=self.bot.get_user(int(lender)).display_name), value=f"${famount}", inline=False)
        embed.add_field(name=t("economy.your_balance", lang), value=f"${new_balance}")
        embed.set_footer(text=t("economy.daily_footer", lang))
        await ctx.send(embed=embed)
        
        # --- Daily Quest System ---
        from src.database.quest import has_daily_quest_today
        if not has_daily_quest_today(int(user_id)):
            try:
                objective = random.choice(DailyQuest.OBJECTIVES)
                start_quest(int(user_id), 'daily_quest', custom_data={
                    'target_stat': objective['stat'],
                    'target_count': objective['count'],
                    'text_key': objective['text_key']
                })
                
                obj_text = t(objective['text_key'], lang, n=objective['count'])
                quest_embed = discord.Embed(
                    title=t("quests.daily_challenge.title", lang),
                    description=t("quests.daily_challenge.new_quest", lang, objective=obj_text),
                    color=discord.Color.blue()
                )
                await ctx.send(embed=quest_embed)
            except Exception as e:
                print(f"Error starting daily quest: {e}")
        
        from src.database.achievement import increment_stat
        increment_stat(int(user_id), "daily_uses", 1)

        unlocks = check_and_unlock_achievements(int(user_id))
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks, lang))

    @commands.command(name='give', aliases=['pay'])
    async def give(self, ctx, recipient: discord.Member, amount: int) -> None:
        """Faire un virement (donner de l'argent)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        sender_id = ctx.author.id
        recipient_id = recipient.id
        if sender_id == recipient_id or amount <= 0:
            return await ctx.send(t("economy.give_invalid", lang))
        if get_balance(sender_id) < amount:
            return await ctx.send(t("economy.give_no_money", lang))
        update_balance(sender_id, -amount)
        update_balance(recipient_id, amount)
        embed = discord.Embed(title=t("economy.give_title", lang), color=discord.Color.green())
        embed.add_field(name=t("economy.sender", lang), value=ctx.author.display_name, inline=True)
        embed.add_field(name=t("economy.receiver", lang), value=recipient.display_name, inline=True)
        embed.add_field(name=t("economy.quantity", lang), value=f"**${amount}**", inline=False)
        await ctx.send(embed=embed)

        unlocks = check_and_unlock_achievements(sender_id)
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks, lang))
        unlocks = check_and_unlock_achievements(recipient_id)
        if unlocks:
            await ctx.send(embed=format_achievements_unlocks(unlocks, lang))
        return None


async def setup(bot):
    await bot.add_cog(Economy(bot))
