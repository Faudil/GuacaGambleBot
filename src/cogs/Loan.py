import discord
from discord.ext import commands
from discord.ui import View, Button

from src.database.balance import get_balance, update_balance
from src.database.loan import create_loan, get_total_debt, get_creditors, repay_debt_logic


class LoanView(View):
    def __init__(self, lender, borrower, amount):
        super().__init__(timeout=60)
        self.lender = lender
        self.borrower = borrower
        self.amount = amount
        self.total_due = int(amount * 1.10)

    @discord.ui.button(label="✍️ Signer le contrat", style=discord.ButtonStyle.success)
    async def accept(self, interaction: discord.Interaction, button: Button):
        if interaction.user != self.borrower:
            return await interaction.response.send_message("Ce n'est pas ta dette !", ephemeral=True)

        res = create_loan(self.lender.id, self.borrower.id, self.amount)

        if res == "SUCCESS":
            await interaction.response.edit_message(
                content=f"🤝 **Accord conclu !**\n{self.lender.mention} a prêté **${self.amount}** à {self.borrower.mention}.\n{self.borrower.mention} devra rembourser **${self.total_due}** (avec intérêts).",
                view=None)
        elif res == "LENDER_BROKE":
            await interaction.response.edit_message(content="❌ Le prêteur n'a plus assez d'argent !", view=None)
        elif res == "DEBT_LIMIT":
            await interaction.response.edit_message(
                content="❌ Limite atteinte : L'emprunteur ne peut pas dépasser **$500** de dettes totales.", view=None)

        self.stop()

    @discord.ui.button(label="Refuser", style=discord.ButtonStyle.danger)
    async def refuse(self, interaction: discord.Interaction, button: Button):
        if interaction.user == self.borrower or interaction.user == self.lender:
            await interaction.response.edit_message(content="❌ Prêt annulé.", view=None)
            self.stop()


class Loan(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name='lend', aliases=['pret'])
    async def lend(self, ctx, borrower: discord.Member, amount: int):
        """Prêter de l'argent avec 10% d'intérêt."""
        if borrower.bot or borrower.id == ctx.author.id:
            return await ctx.send("❌ Tu ne peux pas te prêter à toi-même.")
        if amount <= 0:
            return await ctx.send("❌ Montant invalide.")
        if amount > 500:
            return await ctx.send("❌ Tu ne peux pas prêter plus de 500$ à la même personne.")
        if get_balance(ctx.author.id) < amount:
            return await ctx.send("❌ Tu n'as pas cet argent.")

        total_repay = int(amount * 1.10)

        embed = discord.Embed(title="📜 Contrat de Prêt", color=discord.Color.gold())
        embed.description = (
            f"**Prêteur :** {ctx.author.mention}\n"
            f"**Emprunteur :** {borrower.mention}\n"
            f"**Montant prêté :** ${amount}\n"
            f"**À rembourser :** ${total_repay} (+10%)\n\n"
            f"⚠️ {borrower.mention}, clique pour accepter la dette."
        )

        view = LoanView(ctx.author, borrower, amount)
        return await ctx.send(embed=embed, view=view)

    @commands.command(name='debt', aliases=['dettes'])
    async def my_debt(self, ctx):
        """Voir ce que tu dois aux autres."""
        debt = get_total_debt(ctx.author.id)
        if debt == 0:
            await ctx.send("✅ Tu es libre de toute dette !")
        else:
            debts = get_creditors(ctx.author.id)
            embed = discord.Embed(title="📉 Tes dettes", color=discord.Color.gold())
            for d in debts:
                amount_due = d['amount_due']
                lender_id = d['lender_id']
                embed.add_field(name=self.bot.get_user(lender_id).display_name, value=f"**{amount_due}**", inline=False)
            await ctx.send(f"📉 Tu dois actuellement **${debt}** au total (Max autorisé: $500).")

    @commands.command(name='repay', aliases=['rembourser'])
    async def repay_cmd(self, ctx, amount: int):
        """Rembourser tes dettes. (réparti entre tes différents créanciers)"""
        if amount <= 0: return await ctx.send("Montant invalide.")
        debt = get_total_debt(ctx.author.id)
        if debt == 0:
            return await ctx.send("Tu n'as aucune dette.")
        bal = get_balance(ctx.author.id)
        if bal < amount:
            return await ctx.send("Pas assez d'argent.")
        update_balance(ctx.author.id, -amount)
        paid, details = repay_debt_logic(ctx.author.id, amount)
        change = amount - paid
        if change > 0:
            update_balance(ctx.author.id, change)
        msg = f"💸 Tu as remboursé **${paid}** de dettes."
        if change > 0:
            msg += f" (Je te rends ${change})."
        return await ctx.send(msg)


async def setup(bot):
    await bot.add_cog(Loan(bot))