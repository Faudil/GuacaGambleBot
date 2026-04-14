import discord
from discord.ext import commands
from discord.ui import View, Button

from src.database.balance import get_balance, update_balance
from src.database.loan import create_loan, get_total_debt, get_creditors, repay_debt_logic
from src.database.settings import get_language
from src.utils.i18n import t

class LoanView(View):
    def __init__(self, lender, borrower, amount, lang):
        super().__init__(timeout=60)
        self.lender = lender
        self.borrower = borrower
        self.amount = amount
        self.total_due = int(amount * 1.10)
        self.lang = lang
        
        # Update button labels
        for item in self.children:
            if isinstance(item, Button):
                if item.label == "✍️ Signer le contrat":
                    item.label = t("loan.sign_label", lang)
                elif item.label == "Refuser":
                    item.label = t("loan.refuse_label", lang)

    @discord.ui.button(label="✍️ Signer le contrat", style=discord.ButtonStyle.success)
    async def accept(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.borrower:
            return await interaction.response.send_message(t("loan.not_your_debt", self.lang), ephemeral=True)

        res = create_loan(self.lender.id, self.borrower.id, self.amount)

        if res == "SUCCESS":
            await interaction.response.edit_message(
                content=t("loan.loan_success", self.lang, lender=self.lender.mention, amount=self.amount, borrower=self.borrower.mention, total=self.total_due),
                view=None)
        elif res == "LENDER_BROKE":
            await interaction.response.edit_message(content=t("loan.lender_broke", self.lang), view=None)
        elif res == "DEBT_LIMIT":
            await interaction.response.edit_message(
                content=t("loan.debt_limit", self.lang), view=None)

        self.stop()

    @discord.ui.button(label="Refuser", style=discord.ButtonStyle.danger)
    async def refuse(self, interaction: discord.Interaction, button: Button):
        if interaction.user == self.borrower or interaction.user == self.lender:
            await interaction.response.edit_message(content=t("loan.loan_refused", self.lang), view=None)
            self.stop()


class Loan(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name='lend', aliases=['pret'])
    async def lend(self, ctx, borrower: discord.Member, amount: int):
        """Prêter de l'argent avec 10% d'intérêt."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if borrower.bot or borrower.id == ctx.author.id:
            return await ctx.send(t("loan.no_self_loan", lang))
        if amount <= 0:
            return await ctx.send(t("loan.invalid_amount", lang))
        if amount > 20000:
            return await ctx.send(t("loan.max_loan_limit", lang))
        if get_balance(ctx.author.id) < amount:
            return await ctx.send(t("loan.no_money", lang))

        total_repay = int(amount * 1.10)

        embed = discord.Embed(title=t("loan.contract_title", lang), color=discord.Color.gold())
        embed.description = t("loan.contract_desc", lang, lender=ctx.author.mention, borrower=borrower.mention, amount=amount, total=total_repay)

        view = LoanView(ctx.author, borrower, amount, lang)
        return await ctx.send(embed=embed, view=view)

    @commands.command(name='debt', aliases=['dettes'])
    async def my_debt(self, ctx):
        """Voir ce que tu dois aux autres."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        debt = get_total_debt(ctx.author.id)
        if debt == 0:
            await ctx.send(t("loan.free_of_debt", lang))
        else:
            debts = get_creditors(ctx.author.id)
            embed = discord.Embed(title=t("loan.my_debts_title", lang), color=discord.Color.gold())
            for d in debts:
                amount_due = d['amount_due']
                lender_id = d['lender_id']
                lender = self.bot.get_user(lender_id)
                lender_name = lender.display_name if lender else t("loan.unknown", lang)
                embed.add_field(name=lender_name, value=f"**{amount_due}**", inline=False)
            await ctx.send(t("loan.total_debt_msg", lang, debt=debt), embed=embed)

    @commands.command(name='repay', aliases=['rembourser'])
    async def repay_cmd(self, ctx, amount: int):
        """Rembourser tes dettes. (réparti entre tes différents créanciers)"""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if amount <= 0: return await ctx.send(t("loan.repay_invalid", lang))
        debt = get_total_debt(ctx.author.id)
        if debt == 0:
            return await ctx.send(t("loan.no_debt", lang))
        bal = get_balance(ctx.author.id)
        if bal < amount:
            return await ctx.send(t("loan.no_money_repay", lang))
        update_balance(ctx.author.id, -amount)
        paid, details = repay_debt_logic(ctx.author.id, amount)
        change = amount - paid
        if change > 0:
            update_balance(ctx.author.id, change)
        msg = t("loan.repaid_msg", lang, paid=paid)
        if change > 0:
            msg += t("loan.change_msg", lang, change=change)
        return await ctx.send(msg)


async def setup(bot):
    await bot.add_cog(Loan(bot))