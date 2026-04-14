from discord.ext import commands

from src.database.bank import get_bank_data, deposit_money, withdraw_money
from src.database.settings import get_language
from src.utils.i18n import t


class Bank(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name='deposit', aliases=['dep'])
    async def deposit(self, ctx, amount: str):
        """Dépose de l'argent dans ta banque (max 500)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        wallet, bank = get_bank_data(ctx.author.id)
        if amount.lower() == "all":
            amount_int = wallet
        else:
            try:
                amount_int = int(amount)
            except ValueError:
                return await ctx.send(t("bank.invalid_amount", lang))
        if amount_int <= 0:
            return await ctx.send(t("bank.positive_amount", lang))
        status = deposit_money(ctx.author.id, amount_int)
        if status == "SUCCESS":
            _, new_bank = get_bank_data(ctx.author.id)
            deposited = new_bank - bank
            return await ctx.send(t("bank.deposited", lang, amount=deposited, total=new_bank))
        elif status == "NO_MONEY":
            await ctx.send(t("bank.no_cash", lang))
        elif status == "BANK_FULL":
            await ctx.send(t("bank.bank_full", lang))
        return None

    @commands.command(name='withdraw', aliases=['wd'])
    async def withdraw(self, ctx, amount: str):
        """Retire l'argent de ta banque vers ton portefeuille."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        _, bank = get_bank_data(ctx.author.id)
        if amount.lower() == "all":
            amount_int = bank
        else:
            try:
                amount_int = int(amount)
            except ValueError:
                return await ctx.send(t("bank.invalid_amount", lang))
        if amount_int <= 0:
            return await ctx.send(t("bank.invalid_amount", lang))
        status = withdraw_money(ctx.author.id, amount_int)
        if status:
            return await ctx.send(t("bank.withdrawn", lang, amount=amount_int))
        else:
            return await ctx.send(t("bank.no_bank_money", lang))


async def setup(bot):
    await bot.add_cog(Bank(bot))
