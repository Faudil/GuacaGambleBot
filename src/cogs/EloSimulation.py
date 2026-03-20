import discord
from discord.ext import commands, tasks
from src.database.pets import update_pet_elo, get_random_pet_and_opponent
from src.utils.battle import simulate_battle
import logging

class EloSimulation(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.simulation_loop.start()

    def cog_unload(self):
        self.simulation_loop.cancel()

    @tasks.loop(seconds=15)
    async def simulation_loop(self):
        """Runs background battles between random pets to normalize Elo."""
        try:
            pets = get_random_pet_and_opponent(min_lvl=5, elo_range=500)
            if len(pets) < 2:
                return
            
            pet1, pet2 = pets[0], pets[1]
            
            pet1.heal_full()
            pet2.heal_full()

            await simulate_battle(
                pet1=pet1,
                pet2=pet2,
                send_messages=False,
                sleep_time=0,
                log_size=0
            )
            if pet1.is_alive and not pet2.is_alive:
                result = 1.0
            elif pet2.is_alive and not pet1.is_alive:
                result = 0.0
            else:
                result = 0.5

            diff1, diff2 = pet1.update_elo(pet2, result)
            update_pet_elo(pet1.id, pet1.elo)
            update_pet_elo(pet2.id, pet2.elo)


        except Exception as e:
            logging.error(f"Error in EloSimulation loop: {e}")

    @simulation_loop.before_loop
    async def before_simulation_loop(self):
        await self.bot.wait_until_ready()

async def setup(bot):
    await bot.add_cog(EloSimulation(bot))
