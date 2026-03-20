import discord
from discord.ext import commands

from src.database.db_handler import get_connection
from src.models.Achievement import Achievement

class Achievements(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='achievements', aliases=['succès', 'glory', 'succes'])
    async def view_achievements(self, ctx):
        """Affiche tes succès débloqués et tes points de gloire."""
        user_id = ctx.author.id
        
        conn = get_connection()
        try:
            rows = conn.execute("SELECT achievement_id FROM user_achievements WHERE user_id = ?", (user_id,)).fetchall()
        finally:
            conn.close()
        
        unlocked_ids = [row["achievement_id"] for row in rows]
        unlocked_achievements = [Achievement.get(ach_id) for ach_id in unlocked_ids if Achievement.get(ach_id)]
        
        total_glory = sum(ach.glory for ach in unlocked_achievements)
        
        from src.database.pets import get_all_pet_ranks, RANK_GLORY
        ranks = get_all_pet_ranks()
        pet_glory = sum(RANK_GLORY.get(data["rank"], 0) for data in ranks.values() if data["user_id"] == user_id)
        total_glory += pet_glory
        
        embed = discord.Embed(
            title=f"🏆 Succès de {ctx.author.display_name}", 
            description=f"**Points de Gloire Totaux : {total_glory}** 🌟\n", 
            color=discord.Color.gold()
        )
        
        if pet_glory > 0:
            embed.description += f"*(dont {pet_glory} attribués par le rang de vos familiers)*\n\n"
        else:
            embed.description += "\n"
        
        if not unlocked_achievements:
            embed.description += "Vous n'avez débloqué aucun succès pour le moment."
        else:
            ach_list = []
            for ach in unlocked_achievements:
                ach_list.append(f"{ach.emoji} **{ach.name}** (+{ach.glory} Gloire)\n*{ach.desc}*")
            
            ach_text = "\n\n".join(ach_list)
            if len(ach_text) > 4000:
                ach_text = ach_text[:4000] + "..."
            
            embed.description += ach_text
            
        await ctx.send(embed=embed)

async def setup(bot):
    await bot.add_cog(Achievements(bot))
