from discord.ext import commands

from src.database.bank import get_bank_data, deposit_money, withdraw_money


class Bank(commands.Cog):
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    @commands.command(name='deposit', aliases=['dep'])
    async def deposit(self, ctx, amount: str):
        wallet, bank = get_bank_data(ctx.author.id)
        if amount.lower() == "all":
            amount_int = wallet
        else:
            try:
                amount_int = int(amount)
            except ValueError:
                return await ctx.send("❌ Montant invalide.")
        if amount_int <= 0:
            return await ctx.send("❌ Tu ne peux pas déposer 0 ou moins.")
        status = deposit_money(ctx.author.id, amount_int)
        if status == "SUCCESS":
            _, new_bank = get_bank_data(ctx.author.id)
            deposited = new_bank - bank
            return await ctx.send(f"🏦 Tu as déposé **${deposited}**. Solde banque : **${new_bank}/500**.")
        elif status == "NO_MONEY":
            await ctx.send("❌ Tu n'as pas assez d'argent liquide.")
        elif status == "BANK_FULL":
            await ctx.send("❌ Ta banque est déjà pleine (Max $500) !")
        return None

    @commands.command(name='withdraw', aliases=['wd'])
    async def withdraw(self, ctx, amount: str):
        _, bank = get_bank_data(ctx.author.id)
        if amount.lower() == "all":
            amount_int = bank
        else:
            try:
                amount_int = int(amount)
            except ValueError:
                return await ctx.send("❌ Montant invalide.")
        if amount_int <= 0:
            return await ctx.send("❌ Montant invalide.")
        status = withdraw_money(ctx.author.id, amount_int)
        if status:
            return await ctx.send(f"💸 Tu as retiré **${amount_int}** de la banque.")
        else:
            return await ctx.send("❌ Tu n'as pas assez d'argent en banque.")


async def setup(bot):
    await bot.add_cog(Bank(bot))
